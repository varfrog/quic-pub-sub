package transport

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/quic-go/quic-go"
	"github.com/varfrog/quicpubsub/server/internal/app"
)

// QUICSubscriberConn implements the app.Subscriber for a QUIC connection.
type QUICSubscriberConn struct {
	id         uuid.UUID
	sendStream quic.SendStream
}

var _ app.Subscriber = (*QUICSubscriberConn)(nil)

// NewQUICSubscriberConn is the constructor for QUICSubscriberConn.
func NewQUICSubscriberConn(sendStream quic.SendStream) *QUICSubscriberConn {
	return &QUICSubscriberConn{
		id:         uuid.New(),
		sendStream: sendStream,
	}
}

func (s *QUICSubscriberConn) SendMessageToSubscriber(message []byte) error {
	writtenBytes, err := s.sendStream.Write(message)
	if err != nil {
		return errors.Wrap(err, "sendStream.Write")
	}
	if writtenBytes < len(message) {
		return fmt.Errorf("written %d bytes, message length is %d bytes", writtenBytes, len(message))
	}

	return nil
}

func (s *QUICSubscriberConn) GetID() string {
	return s.id.String()
}
