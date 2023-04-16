package redis

import (
	"context"
	"encoding/json"
	REDIS "github.com/go-redis/redis/v8"
	"reflect"
	"strings"
	"time"
)

type CacheKey struct {
	Key string
	Ttl time.Duration
}

var ChatHistory = &CacheKey{
	Key: "CHAT_HISTORY",
	Ttl: time.Hour,
}

func GetCacheKey(cacheKey *CacheKey, identifiers ...string) *CacheKey {
	joined := strings.Join(identifiers, "_")
	return &CacheKey{
		cacheKey.Key + "_" + joined,
		cacheKey.Ttl,
	}
}

type Redis interface {
	HealthCheck(ctx context.Context) error
	Get(ctx context.Context, cacheKey *CacheKey, t reflect.Type) (interface{}, error)
	Set(ctx context.Context, cacheKey *CacheKey, obj interface{}) error
}

var _ Redis = (*redis)(nil)

type redis struct {
	client *REDIS.Client
}

func (r *redis) HealthCheck(ctx context.Context) error {
	_, err := r.client.Ping(r.client.Context()).Result()
	return err
}

func (r *redis) Get(ctx context.Context, cacheKey *CacheKey, t reflect.Type) (interface{}, error) {
	// TODO: empty check?
	str, err := r.client.Get(ctx, cacheKey.Key).Result()
	if str != "" && err != nil {
		return "", err
	}

	if str == "" {
		str = "[]"
	}

	obj := reflect.New(t).Interface()
	err = json.Unmarshal([]byte(str), obj)
	if err != nil {
		return "", err
	}

	return obj, nil
}

func (r *redis) Set(ctx context.Context, cacheKey *CacheKey, obj interface{}) error {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	_, err = r.client.Set(ctx, cacheKey.Key, bytes, cacheKey.Ttl).Result()
	if err != nil {
		return err
	}

	return nil
}

func GetRedis() (Redis, error) {
	client := REDIS.NewClient(&REDIS.Options{
		Addr: "localhost:6379",
	})
	return &redis{client: client}, nil
}
