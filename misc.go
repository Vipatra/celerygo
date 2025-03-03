package celerygo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"
)

func GetOrigin() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s@%d", hostname, os.Getpid()), nil
}

func Ptr[T any](v T) *T {
	return &v
}

func GetRetryDelayWithExponentialBackOff(currentRetries int64) time.Duration {
	// exponential backoff
	// base * 2 ^ retryCount
	return time.Duration(float64(baseSleepDuration) * math.Pow(float64(2), float64(currentRetries)))
}

func GetBodyTuple(body Body) ([]interface{}, error) {
	var embedMap interface{}
	embedBytes, err := json.Marshal(body.Embed)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(embedBytes, &embedMap)
	if err != nil {
		return nil, err
	}
	return []interface{}{body.Args, body.Kwargs, embedMap}, nil
}

func GetCtxValueString(ctx context.Context, key string) string {
	v := ctx.Value(key)
	stringVal, ok := v.(string)
	if !ok {
		return ""
	}
	return stringVal
}
