# Celery-go

A minimal Celery client for Go that supports publishing tasks to Celery workers. This client follows the Celery 2.0 protocol.

## Features
- Supports task publishing (no task consumer yet)
- Implements Celery 2.0 protocol
- Built-in logging for better observability
- Automatic connection re-establishment with exponential back-off
- Thread-safety

## Installation
```sh
go get github.com/Vipatra/celerygo
```

## Usage

### Import the package
```go
import (
    "github.com/Vipatra/celerygo"
)
```
### Example
```go

package main

import (
"context"
"fmt"
"log"
"sync"
"time"

celerygo "github.com/Vipatra/celerygo"
)

func main() {
	client, err := celerygo.NewCeleryClient("amqp://<username>:<password>@localhost:<port>", "amqp", &celerygo.Options{
		MaxBackOffDuration: 5 * time.Second,
		LogLevel:           celerygo.ErrorLevel,
	})
	if err != nil {
		log.Fatal(err)
	}
	client.RegisterRoutes(map[string]string{
		"long_tasks.*": "long_tasks",
	})

	taskInfo, err := client.SendCeleryTask(context.Background(), "long_tasks.process_portfolio_jobs", nil, map[string]any{
		"org_id": 798798,
	}, &celerygo.AdditionalParameters{
		CountdownSeconds: celerygo.Ptr(30),
	})
	if err != nil {
		log.Println(fmt.Errorf("send task failed: %w", err))
		return
	}
	log.Println(fmt.Sprintf("task info %v",taskInfo))
}
```