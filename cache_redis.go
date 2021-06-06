package main

import (
	"encoding/json"
	"errors"
	"git.kuschku.de/justJanne/bahn-api"
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
		Key:        key,
		Object:     value,
		Expiration: m.expirationTime,
	})
}

func (m RedisCache) Get(key string, value interface{}) error {
	err := m.backend.Get(key, &value)
	if err != nil {
		return err
	} else if value == nil {
		return errors.New("redis returned empty result")
	}
	return nil
}

func NewRedisCache(address string, password string, expirationTime time.Duration) bahn.CacheBackend {
	return RedisCache{
		backend: &cache.Codec{
			Redis: redis.NewClient(&redis.Options{
				Addr:     address,
				Password: password,
			}),
			Marshal:   json.Marshal,
			Unmarshal: json.Unmarshal,
		},
		expirationTime: expirationTime,
	}
}
