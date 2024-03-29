package broadcast

import "sync"

// Channel provides a broadcast channel semantics with closing of subscribers in case of back-pressure.
type Channel[T any] struct {
	lk        sync.Mutex
	listeners []chan<- T
	last      T
}

// Subscribe is used to subscribe to the broadcast channel.
// If the passed channel is full at any point, it will be dropped from subscription and closed.
// To stop subscribing, either the closer function can be used or the channel can be abandoned.
// Passing a channel multiple times to the Subscribe function will result in a panic.
// If there were messages sent previously, it will return it
func (c *Channel[T]) Subscribe(ch chan<- T) (last T, closer func()) {
	c.lk.Lock()
	defer c.lk.Unlock()
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
