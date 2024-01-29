package broadcast

import "sync"

// Channel provides a broadcast channel semantics with closing of subscribers in case of back-pressure.
type Channel[T any] struct {
	lk        sync.Mutex
	listeners []chan<- T
	last      *T
}

// Subscribe is used to subscribe to the broadcast channel.
// If the passed channel is full at any point, it will be dropped from subscription and closed.
// To stop subscribing, either the closer function can be used or the channel can be abandoned.
// Passing a channel multiple times to the Subscribe function will result in a panic.
// If there were messages sent previously, it will return it
func (c *Channel[T]) Subscribe(ch chan<- T) (last *T, closer func()) {
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
				break
			}
		}
		close(ch)
	}
}

func (c *Channel[T]) Last() *T {
	return c.last
}

func (c *Channel[T]) Publish(val T) {
	c.lk.Lock()
	defer c.lk.Unlock()
	var delinquents []chan<- T
	for _, ch := range c.listeners {
		select {
		case ch <- val:
		default:
			delinquents = append(delinquents, ch)
		}
	}
	c.last = &val
	if len(delinquents) == 0 {
		return
	}

	// Remove delinquent channels in place
	n := 0
	for _, ch := range c.listeners {
		isDelinquent := false
		for _, dch := range delinquents {
			if ch == dch {
				isDelinquent = true
				break
			}
		}
		if !isDelinquent {
			c.listeners[n] = ch
			n++
		}
	}
	for _, dch := range delinquents {
		close(dch)
	}
	c.listeners = c.listeners[:n]
}
