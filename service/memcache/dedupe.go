/*
This is an implementation of DeduperStorage interface for memcache.
See the dedupe package for more information.
*/
package memcache

import (
	d "github.com/HailoOSS/service/dedupe"
	mc "github.com/HailoOSS/gomemcache/memcache"
)

type MemcacheDeduper struct {
}

func (f *MemcacheDeduper) Add(key string, value string) error {
	item := &mc.Item{Key: key, Value: []byte(value)}
	err := Add(item)
	if err != nil {
		if err == mc.ErrNotStored {
			return d.ErrorKeyExistsOnWrite
		} else {
			return err
		}
	}
	return nil
}

func (f *MemcacheDeduper) Exists(key string) (bool, error) {
	_, err := Get(key)
	if err != nil {
		return false, d.ErrorUnknown
	}
	return true, nil
}

func (f *MemcacheDeduper) Remove(key string) error {
	err := Delete(key)
	if err != nil {
		return d.ErrorUnknown
	}
	return nil
}
