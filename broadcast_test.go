package broadcast

import (
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubscribeAndPublish(t *testing.T) {
	var ch Channel[int]
	wg := &sync.WaitGroup{}
	c := make(chan int, 1)
	wg.Add(1)

	go func() {
		defer wg.Done()
		assert.Equal(t, <-c, 42, "Expected published value to be received")
	}()

	ch.Subscribe(c)
	ch.Publish(42)
	wg.Wait()
}

func TestChannelClose(t *testing.T) {
	var ch Channel[int]
	c := make(chan int, 1)

	_, closer := ch.Subscribe(c)
	closer()

	_, ok := <-c
	assert.Equal(t, false, ok, "Expected closed channel")
}
func TestMultipleSubscribers(t *testing.T) {
	var ch Channel[int]
	wg := &sync.WaitGroup{}

	// Create 5 subscribers
	for i := 0; i < 5; i++ {
		c := make(chan int, 1)
		wg.Add(1)

		go func() {
			defer wg.Done()
			assert.Equal(t, <-c, 42, "Expected published value to be received")
		}()

		ch.Subscribe(c)
	}

	runtime.Gosched()
	ch.Publish(42)

	// Wait for all subscribers to receive the value
	wg.Wait()
}

func TestMultipleSubscription(t *testing.T) {
	var ch Channel[int]
	c := make(chan int, 1)

	ch.Subscribe(c)
	assert.Panics(t, func() { ch.Subscribe(c) }, "Expected panic on multiple subscription")
}

func TestBackPressure(t *testing.T) {
	var ch Channel[int]
	c := make(chan int, 1)

	ch.Subscribe(c)
	ch.Publish(42)
	ch.Publish(42)

	assert.Equal(t, 42, <-c)
	_, ok := <-c
	assert.Equal(t, false, ok, "Expected channel to be closed due to back pressure")
}

func TestCloser(t *testing.T) {
	var ch Channel[int]
	c := make(chan int, 1)

	_, closer := ch.Subscribe(c)

	ch.Publish(42)

	val, ok := <-c
	assert.True(t, ok, "Expected channel to be open")
	assert.Equal(t, 42, val, "Expected published value to be received")

	closer()

	_, ok = <-c
	assert.False(t, ok, "Expected channel to be closed")
}

func TestBackPressureWithMultipleReceivers(t *testing.T) {
	var ch Channel[int]
	c1 := make(chan int, 1)
	c2 := make(chan int)

	ch.Subscribe(c1)
	_, closer2 := ch.Subscribe(c2)

	ch.Publish(42)

	// Check that the value was received on c1
	val, ok := <-c1
	assert.True(t, ok, "Expected channel to be open")
	assert.Equal(t, 42, val, "Expected published value to be received")

	// Check that c2 was closed due to backpressure
	_, ok = <-c2
	assert.False(t, ok, "Expected channel to be closed due to backpressure")

	// Make sure calling the closer doesn't panic.
	closer2()
}

func TestSubscribeWithLastValue(t *testing.T) {
	var ch Channel[int]
	assert.Zero(t, ch.Last(), "After init Last should be zero value")
	ch.Publish(42)
	assert.NotNil(t, ch.Last(), "After publish Last should not be nil")
	assert.Equal(t, 42, ch.Last(), "After publish Last should be set to correct value")

	last, _ := ch.Subscribe(make(chan int))

	assert.Equal(t, 42, last, "Expected last published value to be received")
}

func TestClose(t *testing.T) {
	var ch Channel[int]
	c1 := make(chan int, 1)

	ch.Subscribe(c1)
	ch.Close()
	_, ok := <-c1
	assert.False(t, ok)
	assert.True(t, ch.IsClosed())
}
