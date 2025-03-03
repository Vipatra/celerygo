package celerygo

import (
	"context"
	"log"
	"testing"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

var (
	client        CeleryClient
	isClientSetup = false
)

func init() {
	client, _ = GetTestCeleryClient()
	isClientSetup = true
}

func GetTestCeleryClient() (CeleryClient, error) {
	client, err := NewCeleryClient("amqp://user:password@localhost:5672", "amqp", nil)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func PurgeQueue(queueName string) {
	client, _ := GetTestCeleryClient()
	conn := client.Connection()
	amqpConn, ok := conn.(*amqp.Connection)
	if !ok {
		log.Fatal("Connection is not of type *amqp.Connection")
	}
	channel, err := amqpConn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	result, err := channel.QueuePurge(queueName, true)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(result)
}

func TestCeleryAMQPClient_SendCeleryTask(t *testing.T) {
	type args struct {
		task             string
		args             []interface{}
		kwArgs           map[string]any
		additionalParams *AdditionalParameters
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Successfully sends tasks to rabbitmq",
			args: args{
				task:             "task.temp",
				args:             nil,
				kwArgs:           nil,
				additionalParams: nil,
			},
			wantErr: false,
		},
		{
			name: "Sends message with ETA using countdown seconds",
			args: args{
				task:   "task.temp",
				args:   nil,
				kwArgs: nil,
				additionalParams: &AdditionalParameters{
					TaskId:           Ptr(uuid.New().String()),
					CountdownSeconds: Ptr(6),
				},
			},
		},
		{
			name: "Sends message with Expiry using expiration",
			args: args{
				task:   "task.temp",
				args:   nil,
				kwArgs: nil,
				additionalParams: &AdditionalParameters{
					TaskId:         Ptr(uuid.New().String()),
					ExpiresSeconds: Ptr(3600),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isClientSetup {
				taskInfo, err := client.SendCeleryTask(context.Background(), tt.args.task, tt.args.args, tt.args.kwArgs, tt.args.additionalParams)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.Equal(t, taskInfo.Message.Headers.Task, tt.args.task)
				}
				if tt.args.additionalParams != nil {
					if tt.args.additionalParams.TaskId != nil {
						assert.Equal(t, *tt.args.additionalParams.TaskId, taskInfo.Id)
					}
					if tt.args.additionalParams.CountdownSeconds != nil {
						assert.NotNil(t, taskInfo.Message.Headers.ETA)
					}
					if tt.args.additionalParams.ExpiresSeconds != nil {
						assert.NotNil(t, taskInfo.Message.Headers.Expires)
					}
				}
			} else {
				log.Println("not setup")
			}
		})
	}
	PurgeQueue("celery")
}
