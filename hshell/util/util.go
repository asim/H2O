package util

import (
	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/protobuf/proto"
	"time"
)

func RemoveDuplicates(xs *[]string) {
	found := make(map[string]bool)
	j := 0
	for i, x := range *xs {
		if !found[x] {
			found[x] = true
			(*xs)[j] = (*xs)[i]
			j++
		}
	}
	*xs = (*xs)[:j]
}

var CacheReload map[string]bool = make(map[string]bool)

func init() {
	cacheExpire()
}

func cacheExpire() {
	go func() {
		for {

			for key := range CacheReload {
				CacheReload[key] = true
			}

			time.Sleep(30 * time.Second)
		}
	}()
}

func SendRequest(req *client.Request, rsp proto.Message) error {
	opts := client.Options{
		"timeout": 5000 * time.Millisecond,
	}
	err := client.Req(req, rsp, opts)
	return err
}

func SendPublication(pub *client.Publication) error {
	err := client.AsyncTopic(pub)
	return err
}
