package configs

import (
	"ChatGPT/configs/redis"
)

type Config interface {
	GetRedis() (redis.Redis, error)
}

var _ Config = (*config)(nil)

type config struct {
	Redis *redis.Redis
}

func (c *config) GetRedis() (redis.Redis, error) {
	if c.Redis != nil {
		return *c.Redis, nil
	}

	r, err := redis.GetRedis()
	if err != nil {
		return nil, err
	}

	c.Redis = &r
	return *c.Redis, nil
}

var GlobalConfig = &config{}
