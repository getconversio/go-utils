package oxr

import (
	"github.com/hashicorp/golang-lru"
)

// LRU cache is an in-memory implementation of the OXR cache interface using
// an LRU as the caching method.
type LRUCache struct {
	lru *lru.ARCCache
}

// Create a new LRU cache.
func NewLRUCache(size int) (*LRUCache, error) {
	cache, err := lru.NewARC(size)
	if err != nil {
		return nil, err
	}
	return &LRUCache{lru: cache}, nil
}

// Implements Cache.Add
func (cache *LRUCache) Add(key string, rates Rates) error {
	cache.lru.Add(key, &rates)
	return nil
}

// Implements Cache.Get
func (cache *LRUCache) Get(key string) (*Rates, error) {
	if rates, ok := cache.lru.Get(key); ok {
		return rates.(*Rates), nil
	}
	return nil, NotFoundError{}
}
