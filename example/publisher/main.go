package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Vipatra/celerygo"
	"github.com/google/uuid"
)

var wg sync.WaitGroup

func main() {
	client, err := celerygo.NewCeleryClient("amqp://user:password@localhost:5672", celerygo.AMQP, &celerygo.Options{
		MaxBackOffDuration: 5 * time.Second,
		LogLevel:           celerygo.InfoLevel,
	})
	if err != nil {
		log.Fatal(err)
	}
	client.RegisterRoutes(map[string]string{
		"long_tasks.*": "long_tasks",
	})

	//publisher.RegisterRoutes(map[string]string{
	//	"long_tasks.*": "long_tasks",
	//})
	//if err != nil {
	//	panic(err)
	//}
	//ticker := time.NewTicker(5 * time.Second)
	//go func() {
	//	for _ = range ticker.C {
	//		log.Println("timeout")
	//		publisher.Close()
	//		ticker.Stop()
	//	}
	//}()

	ctx, _ := context.WithTimeout(context.Background(), 100*time.Minute)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go worker(ctx, client, &wg)
	}
	wg.Wait()
}

func worker(ctx context.Context, client celerygo.CeleryClient, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			log.Println("worker done")
			return
		default:
			time.Sleep(1000 * time.Millisecond)
			log.Println("sending message")
			traceId := uuid.New().String()
			_, err := client.SendCeleryTask(context.Background(), "long_tasks.process_portfolio_jobs", nil, map[string]any{
				"org_id": 798798,
			}, &celerygo.AdditionalParameters{
				CountdownSeconds: celerygo.Ptr(0),
				CorrelationId:    &traceId,
			})
			if err != nil {
				log.Println(fmt.Errorf("send task failed: %w", err))
			}
		}
	}
}
