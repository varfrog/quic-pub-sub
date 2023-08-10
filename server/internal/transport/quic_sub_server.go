package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
	"github.com/quic-go/quic-go"
	"github.com/varfrog/quicpubsub/pkg/quichelper"
	"github.com/varfrog/quicpubsub/server/internal/app"
	"go.uber.org/zap"
	"net"
)

type QUICSubServerConfig struct {
	TLSConfig            *tls.Config
	SubscriberListenPort int // Port to listen on for subscriber connections
	MaxMessageBytes      int // Max bytes to read/write to/from streams, int because io.Reader uses int
}

// QUICSubServer is a server for subscriber connections.
type QUICSubServer struct {
	config         QUICSubServerConfig
	connectionPool *ants.Pool
	observer       *app.Observer
	pinger         *quichelper.Pinger
	logger         *zap.Logger
}

func NewQUICSubServer(
	config QUICSubServerConfig,
	connectionPool *ants.Pool,
	observer *app.Observer,
	pinger *quichelper.Pinger,
	logger *zap.Logger,
) *QUICSubServer {
	return &QUICSubServer{
		config:         config,
		connectionPool: connectionPool,
		observer:       observer,
		pinger:         pinger,
		logger:         logger,
	}
}

func (s *QUICSubServer) AcceptSubscribers(ctx context.Context, quicConfig quic.Config) error {
	listener, err := quic.ListenAddr(
		fmt.Sprintf("127.0.0.1:%d", s.config.SubscriberListenPort),
		s.config.TLSConfig,
		&quicConfig)
	if err != nil {
		return errors.Wrap(err, "quic.ListenAddr")
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping accepting subscriber connections, context cancelled")
			return nil
		default:
			s.logger.Info("Waiting for a subscriber connection")
			conn, err := listener.Accept(ctx)
			if err != nil {
				return errors.Wrap(err, "listener.Accept")
			}
			s.logger.Info("Got a subscriber connection")

			err = s.connectionPool.Submit(func() {
				if err := s.processConnection(context.Background(), conn); err != nil {
					s.logger.Error("processConnection", zap.Error(err))
				}
			})
			if err != nil {
				return errors.Wrap(err, "connectionPool.Submit")
			}
		}
	}
}

func (s *QUICSubServer) processConnection(ctx context.Context, conn quic.Connection) error {
	// Create a cancel function so we can cancel goroutines without cancelling the passed-in context
	ctx, cancel := context.WithCancel(ctx)

	// Create streams for sending messages and receiving events. The calls to open and accept streams
	// are blocking, don't depend on the order these streams are opened or accepted on the peer, so do this
	// in goroutines, and send ready-to-use streams on channels.
	var (
		messagesStreamCh = make(chan quic.SendStream)
		pingStreamCh     = make(chan quic.Stream)
	)
	go func() {
		stream, err := conn.OpenUniStream() // Blocking call
		if err != nil {
			s.logger.Error("OpenUniStream", zap.Error(err))
			cancel()
			return
		}
		s.logger.Info("Message sending stream ready")
		messagesStreamCh <- stream
	}()
	go func() {
		stream, err := conn.AcceptStream(ctx) // Blocking call
		if err != nil {
			s.logger.Error("AcceptStream", zap.Error(err))
			cancel()
			return
		}
		s.logger.Info("Ping stream ready")
		pingStreamCh <- stream
	}()

	// Initialize the subscriber, notify the observer about the new subscriber
	var subscriber *QUICSubscriberConn
	go func() {
		sendStream := <-messagesStreamCh // Wait for the send stream to be available
		s.logger.Info("Subscriber send stream is available")

		subscriber = NewQUICSubscriberConn(sendStream)
		s.logger.Info("Subscriber created", zap.String("id", subscriber.GetID()))

		if err := s.observer.OnSubscriberConnected(subscriber); err != nil {
			s.logger.Warn("OnSubscriberConnected", zap.Error(err))
			cancel()
			return
		}
	}()

	// Receive and respond to pings.
	go quichelper.ReceiveAndRespondToPings(ctx, s.pinger, pingStreamCh, cancel, "Subscriber timed out", s.logger)

	// Monitor failures in other goroutines and on failure notify the observer.
	go func() {
		for {
			select {
			case <-ctx.Done():
				if subscriber != nil {
					err := s.observer.OnSubscriberDisconnected(subscriber)
					if err != nil {
						s.logger.Error("OnSubscriberDisconnected", zap.Error(err))
					}
				}
				return
			}
		}
	}()

	return nil
}

// receivePingMessage waits for a message from the subscriber. Returns ErrNetworkTimeout on timeout, in which case
// we consider the subscriber as disconnected.
func (s *QUICSubServer) receivePingMessage(stream quic.ReceiveStream) error {
	msg := make([]byte, s.config.MaxMessageBytes)
	n, err := stream.Read(msg)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return quichelper.ErrNetworkTimeout
		}
		return errors.Wrap(err, "stream.Read")
	}
	msg = msg[0:n] // Trim to the number of bytes it read
	s.logger.Debug("Received a ping message")

	return nil
}
