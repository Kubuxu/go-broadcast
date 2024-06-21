package broadcast

import (
	"runtime"
	"sync"
)

// Channel provides a broadcast channel semantics with closing of subscribers in case of back-pressure.
type Channel[T any] struct {
	lk        sync.Mutex
	closed    bool
	listeners []chan<- T
	last      T
}

// Subscribe is used to subscribe to the broadcast channel.
// If the passed channel is full at any point, it will be dropped from subscription and closed.
// To stop subscribing, either call the closer function or abandon the channel.
// Passing a channel multiple times to the Subscribe function will result in a panic.
// If there were messages sent previously, the Subscribe will return the last message.
// The default behaviour of subsciber after their channel gets closed should be to create a new
// channel and attempt re-subscibing.
func (c *Channel[T]) Subscribe(ch chan<- T) (last T, closer func()) {
	c.lk.Lock()
	defer c.lk.Unlock()

	if c.closed {
		runtime.Gosched()
		return c.last, func() {}
	}

	for _, exCh := range c.listeners {
		if exCh == ch {
			panic("channel passed multiple times to Subscribe()")
		}
	}
	c.listeners = append(c.listeners, ch)

	return c.last, func() {
		c.lk.Lock()
		defer c.lk.Unlock()
		for i, listener := range c.listeners {
			if listener == ch {
				// Remove the channel from the slice without preserving the order
				c.listeners[i] = c.listeners[len(c.listeners)-1]
				c.listeners = c.listeners[:len(c.listeners)-1]
				close(listener)
				return
			}
		}
	}
}

func (c *Channel[T]) IsClosed() bool {
	c.lk.Lock()
	defer c.lk.Unlock()
	return c.closed
}

// Close will close the broadcast channel, closing all the subscribing channels.
// Note however that the default behaviour of subscribers is to re-try subscribing,
// so ensure that they are closed as well by external
// The primary cause for this function is when the Channel that subscribers attempt to subscribe to
// is getting swapped.
func (c *Channel[T]) Close() {
	c.lk.Lock()
	defer c.lk.Unlock()
	c.closed = true
	for _, listener := range c.listeners {
		close(listener)
	}
	c.listeners = nil
}

func (c *Channel[T]) Last() T {
	c.lk.Lock()
	defer c.lk.Unlock()
	return c.last
}

func (c *Channel[T]) Publish(val T) {
	c.lk.Lock()
	defer c.lk.Unlock()
	for i := 0; i < len(c.listeners); {
		ch := c.listeners[i]
		select {
		case ch <- val:
			i++
		default:
			close(ch)
			// Replace the current channel with the last one and try again.
			lastIdx := len(c.listeners) - 1
			c.listeners[i], c.listeners[lastIdx] = c.listeners[lastIdx], nil
			c.listeners = c.listeners[:lastIdx]
		}
	}

	c.last = val
}
