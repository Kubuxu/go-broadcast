package broadcast

import "sync"

type listener[T any] struct {
	ch    chan<- T
	index int
}

// Channel provides a broadcast channel semantics with closing of subscribers in case of back-pressure.
type Channel[T any] struct {
	lk        sync.Mutex
	listeners []*listener[T]
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
	for _, l := range c.listeners {
		if l.ch == ch {
			panic("channel passed multiple times to Subscribe()")
		}
	}
	l := &listener[T]{ch: ch, index: len(c.listeners)}
	c.listeners = append(c.listeners, l)

	return c.last, func() {
		c.lk.Lock()
		defer c.lk.Unlock()

		// Already removed.
		if l.index < 0 {
			return
		}

		ch := c.listeners[l.index].ch

		lastIdx := len(c.listeners) - 1
		c.listeners[l.index], c.listeners[lastIdx] = c.listeners[lastIdx], nil
		c.listeners = c.listeners[:lastIdx]

		close(ch)
	}
}

func (c *Channel[T]) Last() *T {
	return c.last
}

func (c *Channel[T]) Publish(val T) {
	c.lk.Lock()
	defer c.lk.Unlock()
	for i := 0; i < len(c.listeners); {
		l := c.listeners[i]
		select {
		case l.ch <- val:
			i++
		default:
			// Replace the current channel with the last one and try again.
			lastIdx := len(c.listeners) - 1
			c.listeners[lastIdx].index = i
			c.listeners[i], c.listeners[lastIdx] = c.listeners[lastIdx], nil
			c.listeners = c.listeners[:lastIdx]

			close(l.ch)
			l.index = -1
		}
	}

	c.last = &val
}
