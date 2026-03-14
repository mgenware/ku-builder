package util

type StringCache struct {
	cache map[string]string
}

type StringCacheGetFn func() string

func NewStringCache() *StringCache {
	return &StringCache{
		cache: map[string]string{},
	}
}

func (sc *StringCache) Get(key string, fn StringCacheGetFn) string {
	if val, ok := sc.cache[key]; ok {
		return val
	}
	val := fn()
	sc.cache[key] = val
	return val
}
