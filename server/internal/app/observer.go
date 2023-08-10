package app

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Observer is the internal "event" dispatcher / message broker between connectors (Publishers and Subscribers).
// It accepts events (in form of method calls) and knows how to notify which connectors.
type Observer struct {
	publisherPool  *PublisherPool
	subscriberPool *SubscriberPool
	logger         *zap.Logger
}

// NewObserver is the constructor for Observer.
func NewObserver(
	publisherPool *PublisherPool,
	subscriberPool *SubscriberPool,
	logger *zap.Logger,
) *Observer {
	return &Observer{
		publisherPool:  publisherPool,
		subscriberPool: subscriberPool,
		logger:         logger,
	}
}

func (s *Observer) OnSubscriberConnected(subscriber Subscriber) error {
	if err := s.subscriberPool.Add(subscriber); err != nil {
		return errors.Wrap(err, "add a subscriber to a subscriber pool")
	}

	s.logger.Debug("Subscriber connected", zap.String("subscriber_id", subscriber.GetID()))

	// The server notifies publishers if a subscriber has connected
	for _, publisher := range s.publisherPool.GetAll() {
		if err := publisher.NotifyExistsSubscriber(); err != nil {
			// Don't fail, allow other publishers to receive messages
			s.logger.Warn("publisher.NotifyExistsSubscriber", zap.Error(err))
		}
	}

	return nil
}

func (s *Observer) OnSubscriberDisconnected(subscriber Subscriber) error {
	s.subscriberPool.Remove(subscriber.GetID())

	// If no subscribers are connected, the server must inform the publishers
	if s.subscriberPool.IsEmpty() {
		s.logger.Debug("Last subscriber disconnected, notifying publishers")

		for _, publisher := range s.publisherPool.GetAll() {
			if err := publisher.NotifyNoSubscribers(); err != nil {
				// Don't fail, allow other publishers to receive messages
				s.logger.Warn("publisher.NotifyNoSubscribers", zap.Error(err))
			}
		}
	}

	return nil
}

func (s *Observer) OnPublisherConnected(publisher Publisher) error {
	if err := s.publisherPool.Add(publisher); err != nil {
		return errors.Wrap(err, "add a publisher to a publisher pool")
	}

	// Notify the publisher right away about whether or not there are subscribers
	if !s.subscriberPool.IsEmpty() {
		s.logger.Debug("Subscribers exist, notifying the new publisher")
		if err := publisher.NotifyExistsSubscriber(); err != nil {
			return errors.Wrap(err, "NotifyExistsSubscriber")
		}
	} else {
		s.logger.Debug("No subscribers exist, notifying the new publisher")
		if err := publisher.NotifyNoSubscribers(); err != nil {
			return errors.Wrap(err, "NotifyNoSubscribers")
		}
	}

	return nil
}

func (s *Observer) OnPublisherDisconnected(publisher Publisher) {
	s.logger.Debug("Publisher disconnected", zap.String("publisher_id", publisher.GetID()))
	s.publisherPool.Remove(publisher.GetID())
}

// OnPublisherMessage is called when the server receives a message from a publisher.
func (s *Observer) OnPublisherMessage(message []byte) error {
	s.logger.Debug("Sending message from publisher to subscribers")

	for _, subscriber := range s.subscriberPool.GetAll() {
		if err := subscriber.SendMessageToSubscriber(message); err != nil {
			// Don't fail, allow other publishers to receive messages
			s.logger.Warn("SendMessageToSubscriber", zap.Error(err))
		}
	}
	return nil
}
