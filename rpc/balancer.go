package rpc

import "errors"

var (
	//ErrNoAvailableItem no item is available
	ErrNoAvailableItem = errors.New("no item is available")
)

// Balancer roundrobin instance
type Balancer struct {
	ch chan int

	next  int
	items []interface{}
}

// NewBalancer New balancer instance
func NewBalancer(items []interface{}) *Balancer {
	return &Balancer{ch: make(chan int, 1), items: items}
}

// Pick available item
func (b *Balancer) Pick() (interface{}, error) {
	if len(b.items) == 0 {
		return nil, ErrNoAvailableItem
	}
	b.ch <- 1
	defer func() {
		<-b.ch
	}()

	n := b.next
	r := b.items[n]
	b.next = (b.next + 1) % len(b.items)

	return r, nil
}
