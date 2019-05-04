package main

import (
	"encoding/json"
	"git.kuschku.de/justjanne/bahn-api"
	"time"
)
import "github.com/patrickmn/go-cache"

type MemoryCache struct {
	backend *cache.Cache
}

func (m MemoryCache) Set(key string, value interface{}) error {
	var err error

	var serialized []byte
	if serialized, err = json.Marshal(&value); err != nil {
		return err
	}

	m.backend.SetDefault(key, serialized)

	return nil
}

func (m MemoryCache) Get(key string, value interface{}) error {
	var err error

	var serialized []byte
	if raw, found := m.backend.Get(key); found {
		serialized = raw.([]byte)
	}

	if err = json.Unmarshal(serialized, &value); err != nil {
		return err
	}

	return nil
}

func NewMemoryCache(expirationTime time.Duration) bahn.CacheBackend {
	return MemoryCache{
		backend: cache.New(expirationTime, expirationTime*2),
	}
}