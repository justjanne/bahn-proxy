package main

import (
	"encoding/json"
	"git.kuschku.de/justjanne/bahn-api"
	"github.com/go-redis/cache"
	"github.com/go-redis/redis"
	"time"
)

type RedisCache struct {
	backend        *cache.Codec
	expirationTime time.Duration
}

func (m RedisCache) Set(key string, value interface{}) error {
	return m.backend.Set(&cache.Item{
		Key: key,
		Object: value,
		Expiration: m.expirationTime,
	})
}

func (m RedisCache) Get(key string, value interface{}) error {
	return m.backend.Get(key, &value)
}

func NewRedisCache(expirationTime time.Duration) bahn.CacheBackend {
	return RedisCache{
		backend: &cache.Codec{
			Redis: redis.NewClient(&redis.Options{

			}),
			Marshal:   json.Marshal,
			Unmarshal: json.Unmarshal,
		},
	}
}
