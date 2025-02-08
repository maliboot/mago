package config

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Auth string `yaml:"auth"`
	DB   int    `yaml:"db"`
	Ctx  any

	client *redis.Client
}

func (r *Redis) Client() *redis.Client {
	if r.client == nil {
		r.client = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", r.Host, r.Port),
			Password: r.Auth,
			DB:       r.DB,
		})
	}
	return r.client
}
