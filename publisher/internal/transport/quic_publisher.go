package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/chzyer/logex"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/quic-go/quic-go"
	"github.com/varfrog/quicpubsub/pkg/quichelper"
	"github.com/varfrog/quicpubsub/pkg/sdk"
	"github.com/varfrog/quicpubsub/publisher/internal/app"
	"go.uber.org/zap"
)

type QUICPublisherConfig struct {
	UUID            uuid.UUID
	TLSConfig       *tls.Config
	ServerPort      int
	MaxMessageBytes int
}

// QUICPublisher is the main process of this service.
// It connects to the server and publishes messages as needed.
type QUICPublisher struct {
	config        QUICPublisherConfig
	quicConfig    quic.Config
	messageSender *app.MessageSender
	pinger        *quichelper.Pinger
	logger        *zap.Logger
}

func NewQUICPublisher(
	config QUICPublisherConfig,
	quicConfig quic.Config,
	messageSender *app.MessageSender,
	pinger *quichelper.Pinger,
	logger *zap.Logger,
) *QUICPublisher {
	return &QUICPublisher{
		config:        config,
		quicConfig:    quicConfig,
		messageSender: messageSender,
		pinger:        pinger,
		logger:        logger,
	}
}

func (s *QUICPublisher) Run(ctx context.Context) error {
	// Connect to the server
	s.logger.Info("Connecting to the server")
	conn, err := quic.DialAddr(
		ctx,
		fmt.Sprintf("127.0.0.1:%d", s.config.ServerPort),
		s.config.TLSConfig,
		&s.quicConfig)
	if err != nil {
		return errors.Wrap(err, "transport.DialAddr")
	}
	s.logger.Info("Connected to the server")

	// Create a cancel function for cancelling goroutines created here without cancelling the passed-in ctx
	ctx, cancel := context.WithCancel(ctx)

	// Set up data channels
	var (
		sendMessagesCh    = make(chan bool)  // stops and resumes message sending
		sendMessageFailCh = make(chan error) // receives message sending failures
	)

	// Create streams asynchronously and pass them onto the channels once they become available
	var (
		eventStreamCh   = make(chan quic.ReceiveStream) // For events from the server
		messageStreamCh = make(chan quic.SendStream)    // For messages to the server
		pingStreamCh    = make(chan quic.Stream)
	)
	go func() {
		stream, err := conn.AcceptUniStream(ctx) // Blocking call
		if err != nil {
			s.logger.Error("AcceptUniStream", zap.Error(err))
			cancel()
			return
		}
		s.logger.Info("Event stream ready")
		eventStreamCh <- stream
	}()
	go func() {
		stream, err := conn.OpenUniStream() // Blocking call
		if err != nil {
			s.logger.Error("OpenUniStream", zap.Error(err))
			cancel()
			return
		}
		s.logger.Info("Message sending stream ready")
		messageStreamCh <- stream
	}()
	go func() {
		stream, err := conn.OpenStream() // Blocking call
		if err != nil {
			s.logger.Error("OpenStream", zap.Error(err))
			cancel()
			return
		}
		s.logger.Info("Ping stream ready")
		pingStreamCh <- stream
	}()

	// Listen for events
	go func() {
		stream := <-eventStreamCh // Wait for the stream to be available
		s.logger.Info("Event stream ready, listening for events")
		if err := s.listenForEvents(ctx, stream, sendMessagesCh); err != nil {
			s.logger.Error("listenForEvents", zap.Error(err))
			cancel()
			return
		}
	}()

	// Send messages
	go func() {
		sendStream := <-messageStreamCh // Wait until the stream becomes available
		s.logger.Info("Message stream ready, listening for events")
		s.messageSender.StartLoop(
			ctx,
			NewQUICMessageRecipient(sendStream),
			sendMessagesCh,
			sendMessageFailCh)
	}()

	// Start pinging the server
	go quichelper.SendPings(ctx, s.pinger, pingStreamCh, cancel, s.logger)

	// Monitor for message sending failures
	go s.monitorMsgSendingFailures(ctx, sendMessageFailCh, cancel)

	// Run until we're done
	select {
	case <-ctx.Done():
		logex.Info("Shutting down")
		return nil
	}
}

// listenForEvents continuously receives events like sdk.CodeExistsSubscriber and toggles message
// sending via channel sendMessagesCh.
func (s *QUICPublisher) listenForEvents(
	ctx context.Context,
	eventStreamCh quic.ReceiveStream,
	sendMessagesCh chan<- bool,
) error {
	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("Stopping receiving messages as context is cancelled")
			return nil
		default:
			event, err := quichelper.ReceiveEvent(eventStreamCh, uint64(s.config.MaxMessageBytes))
			if err != nil {
				var unmarshallErr quichelper.UnmarshalError
				if errors.As(err, &unmarshallErr) {
					s.logger.Info("Got corrupt event, ignoring", zap.ByteString("event_body", unmarshallErr.Data))
					continue
				} else if errors.Is(err, quichelper.ErrNetworkTimeout) {
					s.logger.Info("Server timeout, stopping listening for events")
					return nil
				} else {
					return errors.Wrap(err, "ReceiveEvent")
				}
			}
			s.logger.Info("Got event from the server", zap.String("event", event.Code))
			switch event.Code {
			case sdk.CodeExistsSubscriber:
				sendMessagesCh <- true
			case sdk.CodeNoSubscribers:
				sendMessagesCh <- false
			}
		}
	}
}

// monitorMsgSendingFailures waits for an error on channel sendMessageFailCh,
// upon receiving an error, it calls cancel.
func (s *QUICPublisher) monitorMsgSendingFailures(
	ctx context.Context,
	sendMessageFailCh <-chan error,
	cancel context.CancelFunc,
) {
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping monitoring for message sending errors, context cancelled")
			return
		case err := <-sendMessageFailCh:
			s.logger.Error("Failure sending a message", zap.Error(err))
			cancel()
			return
		}
	}
}
