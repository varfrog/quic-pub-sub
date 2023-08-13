package app

//go:generate mockgen -source connectors.go -destination mocks/mock_connectors.go Publisher,Subscriber

// Publisher provides an abstraction for a publisher connection.
type Publisher interface {
	// NotifyExistsSubscriber informs the publisher that there exists at least one subscriber.
	NotifyExistsSubscriber() error

	// NotifyNoSubscribers notifies the publisher that no subscribers are connected.
	NotifyNoSubscribers() error

	// GetID returns a unique identifier for this connection.
	GetID() string
}

// Subscriber provides an abstraction for a subscriber connection.
type Subscriber interface {
	SendMessageToSubscriber(message []byte) error

	// GetID returns a unique identifier for this connection.
	GetID() string
}
