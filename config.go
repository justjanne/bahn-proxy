package main

import "time"

type Config struct {
	Endpoints      EndpointConfig `yaml:"endpoints"`
	Caches         CacheConfig    `yaml:"caches"`
	RequestTimeout time.Duration  `yaml:"request_timeout"`
	MaxResults     int            `yaml:"max_results"`
}

type EndpointConfig struct {
	Iris          string `yaml:"iris"`
	CoachSequence string `yaml:"coach_sequence"`
	Hafas         string `yaml:"hafas"`
}

type CacheConfig struct {
	Redis  RedisCacheConfig  `yaml:"redis"`
	Memory MemoryCacheConfig `yaml:"memory"`
}

type RedisCacheConfig struct {
	Address  string        `yaml:"address"`
	Password string        `yaml:"password"`
	Timeout  time.Duration `yaml:"timeout"`
}

type MemoryCacheConfig struct {
	Timeout time.Duration `yaml:"timeout"`
}
