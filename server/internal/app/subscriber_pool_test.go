package app_test

import (
	. "github.com/onsi/gomega"
	"github.com/varfrog/quicpubsub/server/internal/app"
	mocks "github.com/varfrog/quicpubsub/server/internal/app/mocks"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestSubscriberPool_AddUniqueSubscribers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	g := NewGomegaWithT(t)

	pool := app.NewSubscriberPool()

	mockSubscriber1 := mocks.NewMockSubscriber(ctrl)
	mockSubscriber1.EXPECT().GetID().Return("1")

	mockSubscriber2 := mocks.NewMockSubscriber(ctrl)
	mockSubscriber2.EXPECT().GetID().Return("2")

	err := pool.Add(mockSubscriber1)
	g.Expect(err).To(BeNil())

	err = pool.Add(mockSubscriber2)
	g.Expect(err).To(BeNil())

	g.Expect(pool.GetAll()).To(ConsistOf(mockSubscriber1, mockSubscriber2))
}

func TestSubscriberPool_AddingDuplicateSubscriberDoesNotOverrideTheFirstOne(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	g := NewGomegaWithT(t)

	pool := app.NewSubscriberPool()

	mockSubscriber1 := mocks.NewMockSubscriber(ctrl)
	mockSubscriber1.EXPECT().GetID().Return("1") // Same ID

	mockSubscriber2 := mocks.NewMockSubscriber(ctrl)
	mockSubscriber2.EXPECT().GetID().Return("1") // Same ID

	err := pool.Add(mockSubscriber1)
	g.Expect(err).To(BeNil())

	err = pool.Add(mockSubscriber2)
	g.Expect(err).ToNot(BeNil())

	g.Expect(pool.GetAll()).To(ConsistOf(mockSubscriber1))
}

func TestSubscriberPool_IsEmptyWhenNotEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	g := NewGomegaWithT(t)

	pool := app.NewSubscriberPool()

	mockSubscriber1 := mocks.NewMockSubscriber(ctrl)
	mockSubscriber1.EXPECT().GetID().Return("1")

	err := pool.Add(mockSubscriber1)
	g.Expect(err).To(BeNil())

	g.Expect(pool.IsEmpty()).To(BeFalse())
}

func TestSubscriberPool_IsEmptyWhenEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	g := NewGomegaWithT(t)

	pool := app.NewSubscriberPool()

	g.Expect(pool.IsEmpty()).To(BeTrue())
}

func TestSubscriberPool_Remove(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	g := NewGomegaWithT(t)

	pool := app.NewSubscriberPool()

	mockSubscriber1 := mocks.NewMockSubscriber(ctrl)
	mockSubscriber1.EXPECT().GetID().Return("1")

	err := pool.Add(mockSubscriber1)
	g.Expect(err).To(BeNil())

	pool.Remove("1")

	g.Expect(pool.IsEmpty()).To(BeTrue())
}
