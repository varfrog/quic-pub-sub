package app

import (
	"fmt"
	"sync"
)

// SubscriberPool is a container for subscribers, used to Add or Remove them as they connect or disconnect
// to or from the server.
type SubscriberPool struct {
	subscribers sync.Map
}

// NewSubscriberPool is a constructor for SubscriberPool.
func NewSubscriberPool() *SubscriberPool {
	return &SubscriberPool{
		subscribers: sync.Map{},
	}
}

func (p *SubscriberPool) Add(subscriber Subscriber) error {
	id := subscriber.GetID()
	_, loaded := p.subscribers.LoadOrStore(id, subscriber)
	if loaded {
		return fmt.Errorf("subscriber by ID '%s' exists, not overriding", id)
	}
	return nil
}

func (p *SubscriberPool) Remove(subscriberID string) {
	p.subscribers.Delete(subscriberID)
}

func (p *SubscriberPool) GetAll() []Subscriber {
	var subscribers []Subscriber

	p.subscribers.Range(func(key, value interface{}) bool {
		if subscriber, ok := value.(Subscriber); ok {
			subscribers = append(subscribers, subscriber)
		}
		return true
	})

	return subscribers
}

func (p *SubscriberPool) IsEmpty() bool {
	isEmpty := true
	p.subscribers.Range(func(key, value interface{}) bool {
		isEmpty = false
		return false // Stops iteration
	})
	return isEmpty
}
