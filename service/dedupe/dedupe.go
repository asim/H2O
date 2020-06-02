/*
Package dedupe provides an interface to dedupe anything.
It was originally written for deduping the firehose.
Given a prefix and a size it will create a unique hash
based on the content of the message to be deduped. It will
test if that hash has been seen before in whatever storage
mechanism its been given.

DeduperStorage can be any type of storage that implements
the given interface. See the memcache package for an example.
*/
package dedupe

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"hash/adler32"
	"strconv"
)

/*
Error codes
*/
var (
	ErrorKeyExistsOnWrite = errors.New("Dedupe: key exists while attempting write")
	ErrorKeyExistsOnRead  = errors.New("Dedupe: key exists while attempting read")
	ErrorUnknown          = errors.New("Dedupe: unknown error occured")
	ErrorUnknownType      = errors.New("Dedupe: unknown filtering algorithm")
)

/*
Type - the type of filtering algorithm to use
Size - the of the filter, the bigger the number the more unique (for lossy hashmaps only)
Prefix - some kind of namespace that relates to your implementation
Storage - storage mechanism for searching for duplicates, see memcache package for example
*/
type Deduper struct {
	Type    string
	Size    int
	Prefix  string
	Storage DeduperStorage
}

type DeduperStorage interface {
	Exists(key string) (bool, error)
	Add(key string, value string) error
	Remove(key string) error
}

/*
After implementing your storage interface and instantiating
a Deduper struct, call this method to determine if the message
is a duplicate (true) or is not a duplicate (false). Will return
an error if there is a failure in the storage mechanism.

	d := &Deduper{Type: "hashmap", Size: 1000000, Prefix: "dedupe:firehose:memcache", Storage: &MemcacheDeduper{}}
	if d.Contans(j.Message) == false {
		// process message
	}
*/
func (f *Deduper) Contains(content []byte) (bool, error) {
	if f.Type == "revbloom" {
		return f.revbloom(content)
	}

	if f.Type == "hashmap" {
		return f.hashmap(content)
	}

	return false, nil
}

func (f *Deduper) Remove(content []byte) error {

	key := ""

	if f.Type == "revbloom" {
		key = f.getRevKey(content)
	} else if f.Type == "hashmap" {
		key = f.getHashKey(content)
	}

	if key != "" {
		return f.Storage.Remove(key)
	} else {
		return ErrorUnknownType
	}
}

func (f *Deduper) getRevKey(content []byte) string {
	checksum := int(adler32.Checksum(content))
	index := checksum % f.Size
	key := f.Prefix + strconv.Itoa(index)
	return key
}

func (f *Deduper) getHashKey(content []byte) string {
	h := md5.New()
	h.Write(content)
	hash := hex.EncodeToString(h.Sum(nil))
	key := f.Prefix + hash
	return key
}

/*
Lossy hash map algorithm, reverse bloom filter.
Provides bounded memory usage with a fixed length hash map
Less resource usage but keys can get overwritten
*/
func (f *Deduper) revbloom(content []byte) (bool, error) {
	h := md5.New()
	h.Write(content)
	hash := hex.EncodeToString(h.Sum(nil))

	key := f.getRevKey(content)

	seen := false
	ex, err := f.Storage.Exists(key)
	if err != nil {
		return false, err
	}
	seen = ex
	if seen == false {
		if errr := f.Storage.Add(key, hash); errr != nil {
			// handles race conditions between reads and writes
			if errr == ErrorKeyExistsOnWrite {
				seen = true
			} else {
				return false, errr
			}
		}
	}

	return seen, nil
}

/*
Standard hashmap, using md5 of the content for the key and integer 1 for the value
*/
func (f *Deduper) hashmap(content []byte) (bool, error) {

	key := f.getHashKey(content)

	seen := false
	ex, err := f.Storage.Exists(key)
	if err != nil {
		return false, err
	}

	seen = ex
	if seen == false {
		if errr := f.Storage.Add(key, "1"); errr != nil {
			// handles race conditions between reads and writes
			if errr == ErrorKeyExistsOnWrite {
				seen = true
			} else {
				return false, errr
			}
		}
	}
	return seen, nil

}
