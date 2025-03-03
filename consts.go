package celerygo

import (
	"time"

	"github.com/sirupsen/logrus"
)

const (
	ISO8601                 = "2006-01-02T15:04:05.999999-07:00"
	defaultConsumerLanguage = "py"
	defaultContentType      = "application/json"
	defaultContentEncoding  = "utf-8"
	AMQP                    = "amqp"
	baseSleepDuration       = 100 * time.Millisecond
	defaultChannel          = "celery"
	correlationIdKey        = "trace_id"
)

type LogLevel logrus.Level

const (
	Debug      = LogLevel(logrus.InfoLevel)
	InfoLevel  = LogLevel(logrus.DebugLevel)
	WarnLevel  = LogLevel(logrus.WarnLevel)
	ErrorLevel = LogLevel(logrus.ErrorLevel)
	FatalLevel = LogLevel(logrus.FatalLevel)
	PanicLevel = LogLevel(logrus.PanicLevel)
	TraceLevel = LogLevel(logrus.TraceLevel)
)
