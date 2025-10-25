package main

import (
	"errors"
	"sync"
	"time"
)

// интерфейс кэша
type ICache interface {
	Set(key string, value any) error
	Get(key string) (any, bool)
	Del(key string) bool
	StopCleanup()
}

// internal item
type item struct {
	value any
	ttl   int64 // unix nano
}

type Cache struct {
	mu         sync.RWMutex
	cacheMap   map[string]item
	quit       chan struct{}
	defaultTTL time.Duration
	cleanupInt time.Duration
}

// NewCache создает кэш
func NewCache(defaultTTL, cleanupInterval time.Duration) *Cache {
	c := &Cache{
		cacheMap:   make(map[string]item),
		quit:       make(chan struct{}),
		defaultTTL: defaultTTL,
		cleanupInt: cleanupInterval,
	}
	// очистка
	go func() {
		if cleanupInterval <= 0 {
			return
		}
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.cleanup()
			case <-c.quit:
				return
			}
		}
	}()
	return c
}

func (c *Cache) Set(key string, value any) error {
	if key == "" || value == nil {
		return errors.New("invalid key or value")
	}
	var expiry int64
	if c.defaultTTL > 0 {
		expiry = time.Now().Add(c.defaultTTL).UnixNano()
	}
	c.mu.Lock()
	c.cacheMap[key] = item{value: value, ttl: expiry}
	c.mu.Unlock()
	return nil
}

func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	it, ok := c.cacheMap[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if it.ttl > 0 && time.Now().UnixNano() > it.ttl {
		// expired: delete
		c.mu.Lock()
		delete(c.cacheMap, key)
		c.mu.Unlock()
		return nil, false
	}
	return it.value, true
}

func (c *Cache) Del(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.cacheMap[key]; !ok {
		return false
	}
	delete(c.cacheMap, key)
	return true
}

func (c *Cache) cleanup() {
	now := time.Now().UnixNano()
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, it := range c.cacheMap {
		if it.ttl > 0 && now > it.ttl {
			delete(c.cacheMap, k)
		}
	}
}

// StopCleanup остановит фоновую горутину
func (c *Cache) StopCleanup() {
	close(c.quit)
}
