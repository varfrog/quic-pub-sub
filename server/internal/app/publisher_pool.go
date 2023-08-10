package app

import (
	"fmt"
	"sync"
)

// PublisherPool is a container for publishers, used to Add or Remove them as they connect or disconnect
// to or from the server.
type PublisherPool struct {
	// publishers contains Publisher objects, keys are IDs.
	publishers sync.Map
}

// NewPublisherPool is a constructor for PublisherPool.
func NewPublisherPool() *PublisherPool {
	return &PublisherPool{
		publishers: sync.Map{},
	}
}

func (p *PublisherPool) Add(publisher Publisher) error {
	id := publisher.GetID()
	_, loaded := p.publishers.LoadOrStore(id, publisher)
	if loaded {
		return fmt.Errorf("publisher by ID '%s' exists, not overriding", id)
	}
	return nil
}

func (p *PublisherPool) Remove(publisherID string) {
	p.publishers.Delete(publisherID)
}

func (p *PublisherPool) GetAll() []Publisher {
	var publishers []Publisher

	p.publishers.Range(func(key, value interface{}) bool {
		if publisher, ok := value.(Publisher); ok {
			publishers = append(publishers, publisher)
		}
		return true
	})

	return publishers
}
