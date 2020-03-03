package common

import "sync"

type SConfigCache struct {
	configs map[string]string
	lock sync.RWMutex
}

func NewConfigCache() *SConfigCache {
	return &SConfigCache{configs: make(map[string]string)}
}

func (cc *SConfigCache) Get(key string) (string, bool) {
	cc.lock.RLock()
	defer cc.lock.RUnlock()
	val, ok := cc.configs[key]
	return val, ok
}

func (cc *SConfigCache) Set(key string, val string) {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	cc.configs[key] = val
}

func (cc *SConfigCache) IsExist(key string) bool {
	cc.lock.RLock()
	defer cc.lock.RUnlock()
	_, ok := cc.configs[key]
	return ok
}

func (cc *SConfigCache) BatchSet(configs map[string]string) {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	for k, v := range configs {
		cc.configs[k] = v
	}
}

func (cc *SConfigCache) BatchGet(keys ...string) (vals []string, allOk bool, noKey string) {
	vals = make([]string, 0, len(keys))
	cc.lock.RLock()
	defer cc.lock.RUnlock()
	for _, k := range keys {
		v, ok := cc.configs[k]
		if !ok {
			noKey = k
			return
		}
		vals = append(vals, v)
	}
	allOk = true
	return
}