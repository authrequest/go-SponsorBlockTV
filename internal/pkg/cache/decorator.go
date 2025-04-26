package cache

import (
	"reflect"
	"time"
)

// Decorator wraps a function with caching functionality
type Decorator struct {
	cache    *Cache
	skipArgs int
}

// NewDecorator creates a new cache decorator
func NewDecorator(ttl time.Duration, maxSize int, skipArgs int) *Decorator {
	return &Decorator{
		cache:    NewCache(maxSize, ttl),
		skipArgs: skipArgs,
	}
}

// Cache decorates a function with caching functionality
func (d *Decorator) Cache(fn interface{}) interface{} {
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	// Create a new function that wraps the original
	wrapper := reflect.MakeFunc(fnType, func(args []reflect.Value) []reflect.Value {
		// Generate cache key from arguments
		key := d.generateKey(args[d.skipArgs:])

		// Try to get from cache
		if value, ok := d.cache.Get(key); ok {
			return value.([]reflect.Value)
		}

		// Call original function
		result := fnValue.Call(args)

		// Check if we should ignore TTL (last argument is bool)
		ignoreTTL := false
		if len(args) > 0 {
			if lastArg := args[len(args)-1]; lastArg.Kind() == reflect.Bool {
				ignoreTTL = lastArg.Bool()
			}
		}

		// Store result in cache
		d.cache.Set(key, result, ignoreTTL)

		return result
	})

	return wrapper.Interface()
}

// generateKey creates a cache key from function arguments
func (d *Decorator) generateKey(args []reflect.Value) string {
	key := ""
	for _, arg := range args {
		key += reflect.TypeOf(arg.Interface()).String() + ":" + arg.String() + "|"
	}
	return key
}
