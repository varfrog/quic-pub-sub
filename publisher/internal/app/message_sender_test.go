package app_test

import (
	"context"
	"github.com/varfrog/quicpubsub/publisher/internal/app"
	mocks "github.com/varfrog/quicpubsub/publisher/internal/app/mocks"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"sync"
	"testing"
	"time"
)

func TestMessageSender(t *testing.T) {
	t.Run("Does not send messages until it receives a signal to start", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Setup MessageRecipient
		messageRecipient := mocks.NewMockMessageRecipient(ctrl)
		messageRecipient.EXPECT().SendMessageToRecipient(gomock.Any()).Times(0) // Assertion

		// Setup MessageProvider
		messageProvider := mocks.NewMockMessageProvider(ctrl)
		// GetMessage() shouldn't be called but don't forbid it
		messageProvider.EXPECT().GetMessage().Return([]byte(""), nil).AnyTimes()

		// Setup MessageSender
		sendInterval := time.Millisecond * 50
		messageSender := app.NewMessageSender(messageProvider, sendInterval, zap.NewNop())

		sendToggleCh := make(chan bool)
		failureCh := make(chan error)

		ctx, cancel := context.WithTimeout(context.Background(), sendInterval*5)
		defer cancel()

		go messageSender.StartLoop(ctx, messageRecipient, sendToggleCh, failureCh)
	})

	t.Run("Starts sending messages when told", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Setup MessageRecipient
		messageRecipient := mocks.NewMockMessageRecipient(ctrl)
		messageRecipient.
			EXPECT().
			SendMessageToRecipient([]byte("hello")).
			MinTimes(3) // Ensure it's been called some number of times

		// Setup MessageProvider
		messageProvider := mocks.NewMockMessageProvider(ctrl)
		messageProvider.EXPECT().GetMessage().Return([]byte("hello"), nil).AnyTimes()

		// Setup MessageSender
		sendInterval := time.Millisecond * 50
		messageSender := app.NewMessageSender(messageProvider, sendInterval, zap.NewNop())

		sendToggleCh := make(chan bool)
		failureCh := make(chan error)

		// Run the sender for a lot longer than the send interval
		ctx, cancel := context.WithTimeout(context.Background(), sendInterval*10)
		defer cancel()

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			messageSender.StartLoop(ctx, messageRecipient, sendToggleCh, failureCh)
		}()

		sendToggleCh <- true // Turn on message sending

		// Keep reading the failure channel in order not to block the messenger
		go func() {
			select {
			case <-failureCh:
			}
		}()

		wg.Wait()
	})

	t.Run("Stops sending messages when told", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Setup MessageRecipient
		messageRecipient := mocks.NewMockMessageRecipient(ctrl)
		messageRecipient.
			EXPECT().
			SendMessageToRecipient(gomock.Any()).
			MaxTimes(3) // Ensure max 3 send operations done if we turn off sending after about 1 send op.

		// Setup MessageProvider
		messageProvider := mocks.NewMockMessageProvider(ctrl)
		messageProvider.EXPECT().GetMessage().Return([]byte("hello"), nil)

		// Setup MessageSender
		sendInterval := time.Millisecond * 50
		messageSender := app.NewMessageSender(messageProvider, sendInterval, zap.NewNop())

		sendToggleCh := make(chan bool)
		failureCh := make(chan error)

		// Run 10 times longer than sendInterval
		ctx, cancel := context.WithTimeout(context.Background(), sendInterval*10)
		defer cancel()

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			messageSender.StartLoop(ctx, messageRecipient, sendToggleCh, failureCh)
		}()

		sendToggleCh <- true // Turn on message sending

		// Turn off message sending after about 1 send ops.
		time.Sleep(sendInterval * 1)
		sendToggleCh <- false

		wg.Wait()
	})
}
