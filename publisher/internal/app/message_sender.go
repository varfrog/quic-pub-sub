package app

import (
	"context"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"time"
)

// MessageSender runs a loop that continuously sends messages to a recipient, listens on a channel to stop or resume
// sending.
type MessageSender struct {
	messageProvider MessageProvider
	sendInterval    time.Duration
	logger          *zap.Logger
}

// NewMessageSender is the constructor of MessageSender.
// sendInterval is the wait time between sending messages.
func NewMessageSender(
	messageProvider MessageProvider,
	sendInterval time.Duration,
	logger *zap.Logger,
) *MessageSender {
	return &MessageSender{
		messageProvider: messageProvider,
		sendInterval:    sendInterval,
		logger:          logger,
	}
}

// StartLoop waits on channel "sendMessagesCh" for "true" to start continuously sending messages, for "false" to stop
// sending messages.
// Messages are sent at intervals "sendInterval", configured at construction.
// StartLoop calls "messageFn" to get the message body.
// Notifies channel "failCh" on failure with the error.
func (s *MessageSender) StartLoop(
	ctx context.Context,
	recipient MessageRecipient,
	sendMessagesCh <-chan bool,
	failCh chan<- error,
) {
	send := false
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping sending messages, context cancelled")
			return
		case val := <-sendMessagesCh:
			send = val
		case <-time.After(s.sendInterval):
			if send {
				message, err := s.messageProvider.GetMessage()
				if err != nil {
					failCh <- errors.Wrapf(err, "get message from provider")
				}
				if err := recipient.SendMessageToRecipient(message); err != nil {
					failCh <- errors.Wrapf(err, "send message to recipient")
					return
				}
			}
		}
	}
}
