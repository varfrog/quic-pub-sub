package app

import (
	"fmt"
	"time"
)

// MessageProviderHello implements MessageProvider and provides simple messages
// that say hello, and add the current time and an identifier for this publisher.
type MessageProviderHello struct {
	Identifier string
}

var _ MessageProvider = (*MessageProviderHello)(nil)

func NewMessageProviderHello(identifier string) *MessageProviderHello {
	return &MessageProviderHello{Identifier: identifier}
}

func (s *MessageProviderHello) GetMessage() ([]byte, error) {
	message := fmt.Sprintf(
		"Hello from publisher %s at %s",
		s.Identifier,
		time.Now().Format(time.TimeOnly))

	return []byte(message), nil
}
