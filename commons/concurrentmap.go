package commons

import "sync"

type ConcurrentMap struct {
	m    map[interface{}]interface{}
	lock sync.Mutex
}

func NewConcurrentMap() *ConcurrentMap {
	return &ConcurrentMap{m: make(map[interface{}]interface{})}
}

func (cm *ConcurrentMap) GetValue(key interface{}) (interface{}, bool) {
	defer cm.lock.Unlock()

	cm.lock.Lock()
	v, ok := cm.m[key]

	return v, ok
}

func (cm *ConcurrentMap) SetValue(key, value interface{}) {
	defer cm.lock.Unlock()

	cm.lock.Lock()
	cm.m[key] = value
}

func (cm *ConcurrentMap) DeleteValue(key interface{}) {
	defer cm.lock.Unlock()

	cm.lock.Lock()
	delete(cm.m, key)
}

func (cm *ConcurrentMap) Count() int {
	defer cm.lock.Unlock()

	cm.lock.Lock()
	return len(cm.m)
}

func (cm *ConcurrentMap) Iterate(f func(key, value interface{}, stop *bool)) {
	defer cm.lock.Unlock()

	cm.lock.Lock()
	stop := false
	for k, v := range cm.m {
		f(k, v, &stop)
		if stop {
			return
		}
	}
}
