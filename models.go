package celerygo

import (
	"time"
)

type Message struct {
	Headers   Headers
	BodyTuple []interface{}
}

type Headers struct {
	Lang                string         `json:"lang"`
	Task                string         `json:"task"`
	ID                  string         `json:"id"`
	RootID              string         `json:"root_id"`
	ParentID            *string        `json:"parent_id"`
	Group               *string        `json:"group"`
	Method              *string        `json:"meth,"`
	Shadow              *string        `json:"shadow,"`
	ETA                 *string        `json:"eta,"`
	Expires             *string        `json:"expires,"`
	Retries             *int           `json:"retries,"`
	TimeLimit           *[]interface{} `json:"timelimit,"`
	ArgsRepr            string         `json:"argsrepr,"`
	KWArgsRepr          string         `json:"kwargsrepr,"`
	Origin              string         `json:"origin,"`
	ReplacedTaskNesting *int           `json:"replaced_task_nesting,"`
	// CorrelationId - Added as an additional field to trace the flow of request. By default, celery-go extracts from context value (trace_id). Capitalized to prevent clash with correlation_id of AMQP
	CorrelationId string `json:"CORRELATION_ID"`
}
type Body struct {
	Args   []interface{}          `json:"args"`
	Kwargs map[string]interface{} `json:"kwargs"`
	Embed  Embed                  `json:"embed"`
}

type Embed struct {
	Callbacks []interface{} `json:"callbacks"`
	Errbacks  []interface{} `json:"errbacks"`
	Chain     []interface{} `json:"chain"`
	Chord     interface{}   `json:"chord"`
}

type AdditionalParameters struct {
	// Countdown expects int pointer which specifies after how many seconds
	// the task supposed to be delivered
	CountdownSeconds *int `json:"countdown_second"`
	// Eta expects time string to be in ISO8601 format
	Eta *string `json:"eta"`
	// TaskId is an unique identifier, generally UUID
	TaskId *string `json:"task_id"`
	// ExpiresSeconds task expiry in seconds
	ExpiresSeconds      *int          `json:"expires_seconds"`
	Method              *string       `json:"method"`
	GroupId             *string       `json:"group_id"`
	Group               *string       `json:"group"`
	GroupIndex          *int          `json:"group_index"`
	Retries             int           `json:"retries"`
	Chord               interface{}   `json:"chord"`
	ReplyTo             string        `json:"reply_to"`
	TimeLimit           []interface{} `json:"time_limit"`
	SoftTimeLimit       interface{}   `json:"soft_time_limit"`
	RootId              *string       `json:"root_id"`
	ParentId            *string       `json:"parent_id"`
	RouteName           *string       `json:"route_name"`
	Shadow              *string       `json:"shadow"`
	Chain               []interface{} `json:"chain"`
	Callbacks           []interface{} `json:"callbacks"`
	Errbacks            []interface{} `json:"errbacks"`
	TaskType            *string       `json:"task_type"`
	ReplacedTaskNesting int           `json:"replaced_task_nesting"`
	RoutingKey          *string       `json:"routing_key"`
	Exchange            string        `json:"exchange"`
	Mandatory           bool          `json:"mandatory"`
	Immediate           bool          `json:"immediate"`
	ContentType         *string       `json:"content_type"`
	ContentEncoding     *string       `json:"content_encoding"`
	Language            *string       `json:"language"`
	// CorrelationId - Added as an additional field to trace the flow of task. By default, celery-go extracts from context value (trace_id). Capitalized to prevent clash with correlation_id of AMQP
	CorrelationId *string `json:"CORRELATION_ID"`
}

func (i *AdditionalParameters) GetContentType() string {
	if i.ContentType != nil {
		return *i.ContentType
	}
	return defaultContentType
}

func (i *AdditionalParameters) GetContentEncoding() string {
	if i.ContentEncoding != nil {
		return *i.ContentEncoding
	}
	return defaultContentEncoding
}

func (i *AdditionalParameters) GetConsumerLanguage() string {
	if i.Language != nil {
		return *i.Language
	}
	return defaultConsumerLanguage
}

type TaskInfo struct {
	Id      string  `json:"id"`
	Message Message `json:"message"`
}

type Options struct {
	// MaxBackOffDuration if set connection / channel retry count
	// will be reset to 0 once current retry duration breaches max backoff duration
	MaxBackOffDuration time.Duration
	LogLevel           LogLevel
}

func NewDefaultOptions() *Options {
	return &Options{
		MaxBackOffDuration: time.Minute,
		LogLevel:           InfoLevel,
	}
}
