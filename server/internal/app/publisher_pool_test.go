package app_test

import (
	. "github.com/onsi/gomega"
	"github.com/varfrog/quicpubsub/server/internal/app"
	mocks "github.com/varfrog/quicpubsub/server/internal/app/mocks"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestPublisherPool_AddUniquePublishers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	g := NewGomegaWithT(t)

	pool := app.NewPublisherPool()

	mockPublisher1 := mocks.NewMockPublisher(ctrl)
	mockPublisher1.EXPECT().GetID().Return("1")

	mockPublisher2 := mocks.NewMockPublisher(ctrl)
	mockPublisher2.EXPECT().GetID().Return("2")

	err := pool.Add(mockPublisher1)
	g.Expect(err).To(BeNil())

	err = pool.Add(mockPublisher2)
	g.Expect(err).To(BeNil())

	g.Expect(pool.GetAll()).To(ConsistOf(mockPublisher1, mockPublisher2))
}

func TestPublisherPool_AddingDuplicatePublisherDoesNotOverrideTheFirstOne(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	g := NewGomegaWithT(t)

	pool := app.NewPublisherPool()

	mockPublisher1 := mocks.NewMockPublisher(ctrl)
	mockPublisher1.EXPECT().GetID().Return("1") // Same ID

	mockPublisher2 := mocks.NewMockPublisher(ctrl)
	mockPublisher2.EXPECT().GetID().Return("1") // Same ID

	err := pool.Add(mockPublisher1)
	g.Expect(err).To(BeNil())

	err = pool.Add(mockPublisher2)
	g.Expect(err).ToNot(BeNil())

	g.Expect(pool.GetAll()).To(ConsistOf(mockPublisher1))
}

func TestPublisherPool_Remove(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	g := NewGomegaWithT(t)

	pool := app.NewPublisherPool()

	mockPublisher1 := mocks.NewMockPublisher(ctrl)
	mockPublisher1.EXPECT().GetID().Return("1")

	err := pool.Add(mockPublisher1)
	g.Expect(err).To(BeNil())

	pool.Remove("1")

	g.Expect(pool.GetAll()).To(HaveLen(0))
}
