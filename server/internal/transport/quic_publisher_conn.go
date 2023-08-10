package transport

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/quic-go/quic-go"
	"github.com/varfrog/quicpubsub/pkg/sdk"
	"github.com/varfrog/quicpubsub/server/internal/app"
)

// QUICPublisherConn implements the app.Publisher interface.
type QUICPublisherConn struct {
	id         uuid.UUID
	sendStream quic.SendStream
}

var _ app.Publisher = (*QUICPublisherConn)(nil)

// NewQUICPublisherConn is the constructor for QUICPublisherConn
func NewQUICPublisherConn(sendStream quic.SendStream) *QUICPublisherConn {
	return &QUICPublisherConn{
		id:         uuid.New(),
		sendStream: sendStream,
	}
}

func (s *QUICPublisherConn) NotifyExistsSubscriber() error {
	if err := s.sendEvent(sdk.Event{Code: sdk.CodeExistsSubscriber}); err != nil {
		return errors.Wrap(err, "sendEvent")
	}
	return nil
}

func (s *QUICPublisherConn) NotifyNoSubscribers() error {
	if err := s.sendEvent(sdk.Event{Code: sdk.CodeNoSubscribers}); err != nil {
		return errors.Wrap(err, "sendEvent")
	}
	return nil
}

func (s *QUICPublisherConn) GetID() string {
	return s.id.String()
}

func (s *QUICPublisherConn) sendEvent(event sdk.Event) error {
	messageBytes, err := json.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "json.Marshall")
	}

	writtenBytes, err := s.sendStream.Write(messageBytes)
	if err != nil {
		return errors.Wrap(err, "stream.Write")
	}
	if writtenBytes < len(messageBytes) {
		return fmt.Errorf("written %d bytes, message length is %d bytes", writtenBytes, len(messageBytes))
	}

	return nil
}
