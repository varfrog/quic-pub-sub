package app

//go:generate mockgen -source app.go -destination mocks/app.go

// MessageRecipient describes a recipient to send messages to.
type MessageRecipient interface {
	SendMessageToRecipient(message []byte) error
}

// MessageProvider provides messages to send, to MessageSender.
type MessageProvider interface {
	GetMessage() ([]byte, error)
}
