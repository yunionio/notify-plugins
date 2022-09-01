// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import "sync"

type SConfigCache struct {
	configs map[string]string
	lock    sync.RWMutex
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

func (cc *SConfigCache) Clean() {
	cc.lock.Lock()
	defer cc.lock.Unlock()
	cc.configs = make(map[string]string, len(cc.configs))
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

func (cc *SConfigCache) Map() map[string]string {
	return cc.configs
}
