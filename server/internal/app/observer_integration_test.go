package app_test

import (
	. "github.com/onsi/gomega"
	"github.com/varfrog/quicpubsub/server/internal/app"
	mocks "github.com/varfrog/quicpubsub/server/internal/app/mocks"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"testing"
)

func TestObserver_OnPublisherConnected_subscriberExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	g := NewGomegaWithT(t)

	// Initialize subscribers
	subscriber1 := mocks.NewMockSubscriber(ctrl)
	subscriber1.EXPECT().GetID().AnyTimes().Return("1")

	// Initialize a subscriber pool
	subscriberPool := app.NewSubscriberPool()
	g.Expect(subscriberPool.Add(subscriber1)).To(Succeed())

	// Initialize publishers
	publisher := mocks.NewMockPublisher(ctrl)
	publisher.EXPECT().GetID().AnyTimes().Return("1")
	publisher.EXPECT().NotifyExistsSubscriber().Times(1) // Assertion

	observer := app.NewObserver(app.NewPublisherPool(), subscriberPool, zap.NewNop())
	g.Expect(observer.OnPublisherConnected(publisher)).To(Succeed())
}

func TestObserver_OnPublisherConnected_subscriberNotExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	g := NewGomegaWithT(t)

	// Initialize an empty subscriber pool
	subscriberPool := app.NewSubscriberPool()

	// Initialize publishers
	publisher := mocks.NewMockPublisher(ctrl)
	publisher.EXPECT().GetID().AnyTimes().Return("1")
	publisher.EXPECT().NotifyNoSubscribers().Times(1) // Assertion

	observer := app.NewObserver(app.NewPublisherPool(), subscriberPool, zap.NewNop())
	g.Expect(observer.OnPublisherConnected(publisher)).To(Succeed())
}

func TestObserver_OnPublisherDisconnected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPublisher := mocks.NewMockPublisher(ctrl)
	mockPublisher.EXPECT().GetID().AnyTimes().Return("1")

	observer := app.NewObserver(app.NewPublisherPool(), app.NewSubscriberPool(), zap.NewNop())
	observer.OnPublisherDisconnected(mockPublisher)
}

func TestObserver_OnPublisherMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	g := NewGomegaWithT(t)

	message := []byte("foo")

	// Initialize subscribers
	subscriber1 := mocks.NewMockSubscriber(ctrl)
	subscriber1.EXPECT().GetID().AnyTimes().Return("1")
	subscriber1.EXPECT().SendMessageToSubscriber(message).Times(1) // Assertion

	subscriber2 := mocks.NewMockSubscriber(ctrl)
	subscriber2.EXPECT().GetID().AnyTimes().Return("2")
	subscriber2.EXPECT().SendMessageToSubscriber(message).Times(1) // Assertion

	// Initialize a subscriber pool
	subscriberPool := app.NewSubscriberPool()
	g.Expect(subscriberPool.Add(subscriber1)).To(Succeed())
	g.Expect(subscriberPool.Add(subscriber2)).To(Succeed())

	// Initialize publishers
	publisher := mocks.NewMockPublisher(ctrl)
	publisher.EXPECT().GetID().AnyTimes().Return("1")

	observer := app.NewObserver(app.NewPublisherPool(), subscriberPool, zap.NewNop())
	g.Expect(observer.OnPublisherMessage(message)).To(Succeed())
}

func TestObserver_OnSubscriberConnected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	g := NewGomegaWithT(t)

	// Initialize subscribers
	subscriber := mocks.NewMockSubscriber(ctrl)
	subscriber.EXPECT().GetID().AnyTimes().Return("1")

	// Initialize publishers
	publisher := mocks.NewMockPublisher(ctrl)
	publisher.EXPECT().GetID().AnyTimes().Return("1")
	publisher.EXPECT().NotifyExistsSubscriber().Times(1) // Assertion

	// Initialize pools
	publisherPool := app.NewPublisherPool()
	g.Expect(publisherPool.Add(publisher)).To(Succeed())

	subscriberPool := app.NewSubscriberPool()

	// AcceptPublishers the test
	observer := app.NewObserver(publisherPool, subscriberPool, zap.NewNop())
	g.Expect(observer.OnSubscriberConnected(subscriber)).To(Succeed())
	g.Expect(subscriberPool.IsEmpty()).To(BeFalse())
}

func TestObserver_OnSubscriberDisconnected_LastSubscriberDisconnects(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	g := NewGomegaWithT(t)

	// Initialize publishers
	publisher := mocks.NewMockPublisher(ctrl)
	publisher.EXPECT().GetID().AnyTimes().Return("1")
	publisher.EXPECT().NotifyNoSubscribers().Times(1) // Assertion

	// Initialize subsribers
	mockSubscriber := mocks.NewMockSubscriber(ctrl)
	mockSubscriber.EXPECT().GetID().AnyTimes().Return("1")

	publisherPool := app.NewPublisherPool()
	g.Expect(publisherPool.Add(publisher)).To(Succeed())

	observer := app.NewObserver(publisherPool, app.NewSubscriberPool(), zap.NewNop())
	g.Expect(observer.OnSubscriberDisconnected(mockSubscriber)).To(Succeed())
}
