package localisation

import (
	"fmt"
	"sync"
	"testing"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"

	"github.com/HailoOSS/platform/server"
)

func TestCachingHobDiscriminator(t *testing.T) {
	times := 0
	testDiscriminator := func(req *server.Request) string {
		times++
		return fmt.Sprintf("result-%d", times)
	}

	cachedDiscriminator := cachingHobDiscriminator(testDiscriminator)
	testReq := server.NewRequestFromDelivery(amqp.Delivery{
		MessageId: "1",
	})

	// Test basic caching
	for i := 0; i < 1000; i++ {
		assert.Equal(t, "result-1", cachedDiscriminator(testReq), "Wasn't cached :(")
	}

	// Test concurrently
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.Equal(t, "result-1", cachedDiscriminator(testReq), "Concurrent caching is fucked :(")
		}()
	}
	wg.Wait()

	// Check things get pushed out after 300
	for i := 0; i < 300; i++ {
		cachedDiscriminator(server.NewRequestFromDelivery(amqp.Delivery{
			MessageId: fmt.Sprintf("foo-%d", i),
		}))
	}

	assert.NotEqual(t, "result-1", cachedDiscriminator(testReq), "LRU isn't capped at 300 results")
}

// @TODO: Need to properly test the actual authorisers, but this isn't possible at the moment given there's no request
// mocking functionality
