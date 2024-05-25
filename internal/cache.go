package internal

import (
	"fmt"
	"sync"
	"time"
)

type cache struct {
	Data            map[string]cacheNode `json:"data"`
	Capacity        uint32               `json:"capacity"`
	CumulativeBytes uint32               `json:"cumulativeBytes"`
	NodeTimeToLive  uint16               `json:"nodeTimeToLive"`
	NodeSize        uint16               `json:"nodeSize"`
	LimitInBytes    uint32               `json:"limitInBytes"`
	Mutex           sync.RWMutex         `json:"-"`
}

type cacheNode struct {
	Value  string    `json:"value"`
	Expiry time.Time `json:"expiry"`
}

func (cache *cache) Set(key, value string) error {
	now := time.Now()
	cacheNode := cacheNode{
		Value:  value,
		Expiry: now.Add(300 * time.Second),
	}

	incomingDataByteSize := len(key) + len(value)

	if incomingDataByteSize > int(cache.NodeSize) {
		return fmt.Errorf("node byte limit exceeded. Max is: %d, got %d", cache.NodeSize)
	}

	if incomingDataByteSize+int(cache.CumulativeBytes) > int(cache.LimitInBytes) {
		return fmt.Errorf("cache byte limit exceeded. Max is: %d", cache.LimitInBytes)
	} else {
		cache.CumulativeBytes = uint32(incomingDataByteSize) + cache.CumulativeBytes
	}

	cache.Mutex.Lock()
	defer cache.Mutex.Unlock()

	delete(cache.Data, key)
	cache.Data[key] = cacheNode

	if len(cache.Data) > int(cache.Capacity) {
		// delete first key
		for keyToBeDeleted := range cache.Data {
			delete(cache.Data, keyToBeDeleted)
			break
		}
	}
	return nil
}

func (cache *cache) Get(key string) (string, bool) {
	cache.Mutex.RLock()
	defer cache.Mutex.RUnlock()

	node, ok := cache.Data[key]

	if node.Expiry.Before(time.Now()) {
		delete(cache.Data, key)
		return "", true
	}

	return node.Value, ok
}

func (cache *cache) Delete(key string) {
	cache.Mutex.Lock()
	defer cache.Mutex.Unlock()

	delete(cache.Data, key)
}
