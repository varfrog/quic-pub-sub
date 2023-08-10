package transport

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/quic-go/quic-go"
	"github.com/varfrog/quicpubsub/publisher/internal/app"
)

// QUICMessageRecipient implements app.MessageRecipient.
type QUICMessageRecipient struct {
	stream quic.SendStream
}

var _ app.MessageRecipient = (*QUICMessageRecipient)(nil)

func NewQUICMessageRecipient(stream quic.SendStream) *QUICMessageRecipient {
	return &QUICMessageRecipient{stream: stream}
}

func (s *QUICMessageRecipient) SendMessageToRecipient(message []byte) error {
	writtenBytes, err := s.stream.Write(message)
	if err != nil {
		return errors.Wrap(err, "stream.Write")
	}
	if writtenBytes < len(message) {
		return fmt.Errorf("written only %d bytes, message length is %d bytes", writtenBytes, len(message))
	}
	return nil
}
