package cache

import (
	"sync"
	"time"
)

// CacheItem 缓存项
type CacheItem struct {
	Value      interface{}
	ExpireTime time.Time
}

// LRUCache 带 TTL 的 LRU 缓存
type LRUCache struct {
	items map[string]*CacheItem
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewLRUCache 创建新的 LRU 缓存
func NewLRUCache(ttl time.Duration) *LRUCache {
	cache := &LRUCache{
		items: make(map[string]*CacheItem),
		ttl:   ttl,
	}

	// 启动清理 goroutine
	go cache.cleanup()

	return cache
}

// Get 获取缓存项
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// 检查是否过期
	if time.Now().After(item.ExpireTime) {
		return nil, false
	}

	return item.Value, true
}

// Set 设置缓存项
func (c *LRUCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &CacheItem{
		Value:      value,
		ExpireTime: time.Now().Add(c.ttl),
	}
}

// Delete 删除缓存项
func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear 清空缓存
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*CacheItem)
}

// Size 返回缓存大小
func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// cleanup 定期清理过期项
func (c *LRUCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.ExpireTime) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
