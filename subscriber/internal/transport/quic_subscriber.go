package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/chzyer/logex"
	"github.com/pkg/errors"
	"github.com/quic-go/quic-go"
	"github.com/varfrog/quicpubsub/pkg/quichelper"
	"go.uber.org/zap"
	"net"
)

type QUICSubscriberConfig struct {
	TLSConfig       *tls.Config
	ServerPort      int
	MaxMessageBytes int
}

// QUICSubscriber is the main process of this service.
// It connects to the server and receives messages.
type QUICSubscriber struct {
	config     QUICSubscriberConfig
	quicConfig quic.Config
	pinger     *quichelper.Pinger
	logger     *zap.Logger
}

func NewQUICSubscriber(
	config QUICSubscriberConfig,
	quicConfig quic.Config,
	pinger *quichelper.Pinger,
	logger *zap.Logger,
) *QUICSubscriber {
	return &QUICSubscriber{
		config:     config,
		quicConfig: quicConfig,
		pinger:     pinger,
		logger:     logger,
	}
}

func (s *QUICSubscriber) Run(ctx context.Context) error {
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

	// Create streams for sending messages and receiving events. The calls to open and accept streams
	// are blocking, don't depend on the order these streams are opened or accepted on the peer, so do this
	// in goroutines, and send ready-to-use streams on channels.
	var (
		messagesStreamCh = make(chan quic.ReceiveStream)
		pingStreamCh     = make(chan quic.Stream)
	)
	go func() {
		stream, err := conn.AcceptUniStream(ctx) // Blocking call
		if err != nil {
			s.logger.Error("AcceptUniStream", zap.Error(err))
			cancel()
			return
		}
		s.logger.Info("Message stream ready")
		messagesStreamCh <- stream
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

	// Start pinging the server
	go quichelper.SendPings(ctx, s.pinger, pingStreamCh, cancel, s.logger)

	// Receive messages from the server
	go func() {
		stream := <-messagesStreamCh // Wait until the stream becomes available
		s.logger.Info("Receiving messages")
		if err := s.listenForMessages(ctx, stream); err != nil {
			s.logger.Error("listenForMessages", zap.Error(err))
			cancel()
			return
		}
	}()

	// Run until we're done
	select {
	case <-ctx.Done():
		logex.Info("Shutting down")
		return nil
	}
}

// listenForMessages continuously reads the givem stream and outputs messages it receives.
func (s *QUICSubscriber) listenForMessages(ctx context.Context, stream quic.ReceiveStream) error {
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping receiving messages as context is cancelled")
			return nil
		default:
			msg, err := s.receiveMessage(stream)
			if err != nil {
				if errors.Is(err, quichelper.ErrNetworkTimeout) {
					s.logger.Info("Server timeout, stopping listening for events")
					return nil
				}
				return errors.Wrap(err, "receiveMessage")
			}
			s.logger.Info("Received message", zap.ByteString("msg", msg))
		}
	}
}

// receiveMessage reads the stream, returns quichelper.ErrNetworkTimeout on timeout.
func (s *QUICSubscriber) receiveMessage(stream quic.ReceiveStream) ([]byte, error) {
	msg := make([]byte, s.config.MaxMessageBytes)
	n, err := stream.Read(msg)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil, quichelper.ErrNetworkTimeout
		}
		return nil, errors.Wrap(err, "stream.Read")
	}
	msg = msg[0:n] // Trim to the number of bytes it read

	return msg, nil
}
