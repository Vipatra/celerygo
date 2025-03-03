package celerygo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

var mutex sync.Mutex

type CeleryClient interface {
	Close() error
	RegisterRoutes(routes map[string]string)
	SendCeleryTask(ctx context.Context, task string, args []interface{}, kwArgs map[string]any, additionalParams *AdditionalParameters) (*TaskInfo, error)
	Connection() interface{}
}

type CeleryAMQPClient struct {
	conn                     *amqp.Connection
	ampqChannel              *amqp.Channel
	routes                   map[string]string
	currentChannelRetries    atomic.Int64
	currentConnectionRetries atomic.Int64
	options                  *Options
}

// NewCeleryClient - creates new celery publisher for given broker
// connects to broker and attaches connection to celeryClient object
func NewCeleryClient(dsn string, broker string, options *Options) (CeleryClient, error) {
	clientOptions := options
	if clientOptions == nil {
		clientOptions = NewDefaultOptions()
	}
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.Level(clientOptions.LogLevel))
	if strings.EqualFold(broker, AMQP) {
		logrus.Info("Using AMQP broker")
		amqpClient := &CeleryAMQPClient{}
		amqpClient.options = clientOptions
		err := amqpClient.connect(dsn)
		if err != nil {
			return nil, err
		}
		return amqpClient, nil
	}
	return nil, errors.New(fmt.Sprintf("Celery publisher not supported for broker : %s", broker))
}

// Channel - returns the ampqChannel if already exists and is not closed
// otherwise returns error
func (c *CeleryAMQPClient) Channel() (*amqp.Channel, error) {
	if c.ampqChannel != nil {
		if !c.ampqChannel.IsClosed() {
			return c.ampqChannel, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("Celery AMQP channel not open"))
}

// Connection - returns connection
func (c *CeleryAMQPClient) Connection() interface{} {
	return c.conn
}

// createChannel - creates a new channel with disconnection listener
// this will indefinitely try to acquire the channel in case of error
func (c *CeleryAMQPClient) createChannel() error {
	logrus.Infof("creating new channel")
	for {
		amqpChannel, err := c.conn.Channel()
		if err != nil {
			logrus.Errorf("rrror creating amqpChannel : %s", err)
			delay := GetRetryDelayWithExponentialBackOff(c.currentChannelRetries.Load())
			c.currentChannelRetries.Add(1)
			if delay > c.options.MaxBackOffDuration {
				logrus.Infof("resetting the channel retries")
				// for next cycle reset the retries
				c.resetChannelRetries()
			}
			logrus.Infof("Retrying in %v", delay)
			time.Sleep(delay)
			continue
		}
		logrus.Infof("attaching the channel")
		// attach AMQP channel
		c.ampqChannel = amqpChannel
		// listens to channel errors and calls createChannel recursively
		go c.handleChannelDisconnects(amqpChannel, err)
		// reset retries once channel is established
		c.resetChannelRetries()
		return nil
	}
}

// connect - create new connection with AMQP
// indefinitely tries to get the connection in case of error
// spawns a listener to check if connection errors to reconnect automatically
func (c *CeleryAMQPClient) connect(dsn string) error {
	mutex.Lock()
	defer mutex.Unlock()
	logrus.Infof("attempting to connect to RabbitMQ")
	for {
		conn, err := amqp.Dial(dsn)
		if err != nil {
			logrus.Infof("Failed to connect to RabbitMQ: %v", err)
			delay := GetRetryDelayWithExponentialBackOff(c.currentConnectionRetries.Load())
			c.currentConnectionRetries.Add(1)
			if delay > c.options.MaxBackOffDuration {
				// for next cycle reset the retries
				c.resetConnectionRetries()
			}
			logrus.Infof("Retrying connection establishment in %v", delay)
			time.Sleep(delay)
			continue
		}
		logrus.Info("Connected to RabbitMQ")
		// attach AMQP connection
		c.conn = conn
		c.resetConnectionRetries()
		err = c.createChannel()
		if err != nil {
			logrus.Errorf("Failed to create ampqChannel to RabbitMQ: %s", err)
			continue
		}
		// listens for connection errors and calls connect() to re-connect to AMQP
		go c.handleConnectionDisconnects(conn, err, dsn)
		// reset retries once connection is established
		return nil
	}
}

func (c *CeleryAMQPClient) handleChannelDisconnects(amqpChannel *amqp.Channel, err error) {
	_, ok := <-amqpChannel.NotifyClose(make(chan *amqp.Error))
	if !ok {
		logrus.Info("connection gracefully closed, not attempting to connect to RabbitMQ")
		return
	}
	err = c.createChannel()
	if err != nil {
		logrus.Errorf("Failed to recreate amqpChannel to RabbitMQ: %s", err)
		return
	}
}

func (c *CeleryAMQPClient) handleConnectionDisconnects(conn *amqp.Connection, err error, dsn string) {
	_, ok := <-conn.NotifyClose(make(chan *amqp.Error))
	if !ok {
		logrus.Info("connection gracefully closed")
		return
	}
	err = c.connect(dsn)
	if err != nil {
		logrus.Infof("Failed to reconnect to RabbitMQ: %s", err)
		return
	}
}

func (c *CeleryAMQPClient) Close() error {
	err := c.conn.Close()
	if err != nil {
		return err
	}
	return c.ampqChannel.Close()
}

func (c *CeleryAMQPClient) SendCeleryTask(ctx context.Context, task string, args []interface{}, kwArgs map[string]any, additionalParams *AdditionalParameters) (*TaskInfo, error) {
	var amqpHeaders amqp.Table
	if additionalParams == nil {
		additionalParams = &AdditionalParameters{}
	}
	message, err := c.prepareMessage(ctx, task, args, kwArgs, additionalParams)
	if err != nil {
		logrus.Infof("Failed to prepare message : %s", err)
		return nil, err
	}
	bodyTupleBytes, err := json.Marshal(message.BodyTuple)
	if err != nil {
		logrus.Infof("Failed to marshal body tuple : %s", err)
		return nil, err
	}
	headerBytes, err := json.Marshal(message.Headers)
	if err != nil {
		logrus.Infof("Failed to marshal headers tuple : %s", err)
		return nil, err
	}
	err = json.Unmarshal(headerBytes, &amqpHeaders)
	if err != nil {
		logrus.Infof("Failed to unmarshal headers tuple : %s", err)
		return nil, err
	}
	routingKey := c.getRoutingKeyForTask(task)
	err = c.ampqChannel.PublishWithContext(ctx, additionalParams.Exchange, routingKey, additionalParams.Mandatory, additionalParams.Immediate, amqp.Publishing{
		Headers:         amqpHeaders,
		Body:            bodyTupleBytes,
		ContentType:     additionalParams.GetContentType(),
		ContentEncoding: additionalParams.GetContentEncoding(),
		CorrelationId:   message.Headers.ID,
		ReplyTo:         additionalParams.ReplyTo,
	})
	return &TaskInfo{
		Id:      message.Headers.ID,
		Message: *message,
	}, err
}

func (c *CeleryAMQPClient) RegisterRoutes(routes map[string]string) {
	c.routes = routes
}

// prepareMessage - Prepares message based on celery 2.0 protocol
// If taskId is supplied
func (*CeleryAMQPClient) prepareMessage(ctx context.Context, task string, args []interface{}, kwArgs map[string]any, params *AdditionalParameters) (*Message, error) {
	var taskId string
	var expiry *string = nil
	var correlationId string

	if params == nil {
		params = &AdditionalParameters{}
	}
	if args == nil {
		args = []interface{}{}
	}
	if kwArgs == nil {
		kwArgs = map[string]any{}
	}
	argsBytes, err := json.Marshal(args)
	if err != nil {
		logrus.Errorf("Failed to marshal args to json : %s", err)
		return nil, err
	}
	kwArgBytes, err := json.Marshal(kwArgs)
	if err != nil {
		logrus.Errorf("Failed to marshal kwArgs to json : %s", err)
		return nil, err
	}
	if params.TaskId != nil {
		taskId = *params.TaskId
	} else {
		taskId = uuid.New().String()
	}

	origin, err := GetOrigin()
	if err != nil {
		return nil, err
	}

	if params.CountdownSeconds != nil {
		now := time.Now().UTC()
		params.Eta = Ptr(now.Add(time.Duration(*params.CountdownSeconds) * time.Second).Format(ISO8601))
	}

	if params.ExpiresSeconds != nil {
		now := time.Now().UTC()
		expiry = Ptr(now.Add(time.Duration(*params.ExpiresSeconds) * time.Second).Format(ISO8601))
	}

	if params.CorrelationId != nil {
		correlationId = *params.CorrelationId
	} else {
		correlationId = GetCtxValueString(ctx, CorrelationIdKey)
	}

	headers := Headers{
		Lang:                params.GetConsumerLanguage(),
		Task:                task,
		ID:                  taskId,
		RootID:              taskId,
		ParentID:            params.ParentId,
		Group:               params.Group,
		Method:              params.Method,
		Shadow:              params.Shadow,
		ETA:                 params.Eta,
		Expires:             expiry,
		Retries:             Ptr(params.Retries),
		TimeLimit:           params.TimeLimit,
		ArgsRepr:            string(argsBytes),
		KWArgsRepr:          string(kwArgBytes),
		Origin:              origin,
		ReplacedTaskNesting: Ptr(params.ReplacedTaskNesting),
		CorrelationId:       correlationId,
	}

	body := Body{
		Args:   args,
		Kwargs: kwArgs,
		Embed: Embed{
			Callbacks: params.Callbacks,
			Errbacks:  params.Errbacks,
			Chain:     params.Chain,
			Chord:     params.Chord,
		},
	}
	bodyTuple, err := GetBodyTuple(body)
	if err != nil {
		return nil, err
	}
	msg := Message{
		Headers:   headers,
		BodyTuple: bodyTuple,
	}
	return &msg, nil
}

func (c *CeleryAMQPClient) getRoutingKeyForTask(taskName string) string {
	for prefix, queueName := range c.routes {
		if strings.HasPrefix(taskName, strings.TrimSuffix(prefix, ".*")) {
			return queueName
		}
	}
	return defaultChannel
}

func (c *CeleryAMQPClient) resetChannelRetries() {
	c.currentChannelRetries.Store(0)
}

func (c *CeleryAMQPClient) resetConnectionRetries() {
	c.currentConnectionRetries.Store(0)
}
