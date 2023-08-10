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
	"time"
)

type QUICPubServerConfig struct {
	TLSConfig           *tls.Config
	PublisherListenPort int           // Port to listen on for publisher connections
	MaxMessageBytes     int           // Max bytes to read/write to/from streams, int because io.Reader uses int
	PingTimeout         time.Duration // Ping request read/write deadline exceeding which the peer is considered dead
}

// QUICPubServer is a server for publishers connections.
type QUICPubServer struct {
	config         QUICPubServerConfig
	connectionPool *ants.Pool
	observer       *app.Observer
	pinger         *quichelper.Pinger
	logger         *zap.Logger
}

func NewQUICPubServer(
	config QUICPubServerConfig,
	connectionPool *ants.Pool,
	observer *app.Observer,
	pinger *quichelper.Pinger,
	logger *zap.Logger,
) *QUICPubServer {
	return &QUICPubServer{
		config:         config,
		connectionPool: connectionPool,
		observer:       observer,
		pinger:         pinger,
		logger:         logger,
	}
}

func (s *QUICPubServer) AcceptPublishers(ctx context.Context, quicConfig quic.Config) error {
	listener, err := quic.ListenAddr(
		fmt.Sprintf("127.0.0.1:%d", s.config.PublisherListenPort),
		s.config.TLSConfig,
		&quicConfig)
	if err != nil {
		return errors.Wrap(err, "quic.ListenAddr")
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping accepting publisher connections, context cancelled")
			return nil
		default:
			s.logger.Info("Waiting for publisher connections")
			conn, err := listener.Accept(ctx)
			if err != nil {
				return errors.Wrap(err, "listener.Accept")
			}
			s.logger.Info("Got a publisher connection")

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

func (s *QUICPubServer) processConnection(ctx context.Context, conn quic.Connection) error {
	// Create a cancel function so we can cancel goroutines without cancelling the passed-in context
	ctx, cancel := context.WithCancel(ctx)

	// Create streams for sending messages and receiving events. The calls to open and accept streams
	// are blocking, don't depend on the order these streams are opened or accepted on the peer, so do this
	// in goroutines, and send ready-to-use streams on channels.
	var (
		eventStreamCh   = make(chan quic.SendStream)    // For events to publishers
		messageStreamCh = make(chan quic.ReceiveStream) // For messages from publishers
		pingStreamCh    = make(chan quic.Stream)
	)
	go func() {
		stream, err := conn.OpenUniStream() // Blocking call
		if err != nil {
			s.logger.Error("OpenUniStream", zap.Error(err))
			cancel()
			return
		}
		s.logger.Info("Event sending stream ready")
		eventStreamCh <- stream
	}()
	go func() {
		stream, err := conn.AcceptUniStream(ctx) // Blocking call
		if err != nil {
			s.logger.Error("AcceptUniStream", zap.Error(err))
			cancel()
			return
		}
		s.logger.Info("Message stream ready")
		messageStreamCh <- stream
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

	// Initialize the publisher, notify the observer about a new publisher
	var publisher *QUICPublisherConn
	go func() {
		stream := <-eventStreamCh // Wait for the send stream to be available
		s.logger.Info("Publisher send stream is available")

		publisher = NewQUICPublisherConn(stream)
		s.logger.Info("Publisher created", zap.String("id", publisher.GetID()))

		if err := s.observer.OnPublisherConnected(publisher); err != nil {
			s.logger.Error("Failure in OnPublisherConnected", zap.Error(err))
			cancel()
			return
		}
	}()

	// Receive messages.
	go func() {
		stream := <-messageStreamCh // Wait until the stream becomes available
		s.logger.Debug("Message stream available")
		if err := s.receiveMessages(ctx, stream); err != nil {
			s.logger.Error("receiveMessages", zap.Error(err))
			cancel()
			return
		}
	}()

	go quichelper.ReceiveAndRespondToPings(ctx, s.pinger, pingStreamCh, cancel, "Publisher timed out", s.logger)

	go s.monitorDoneContext(ctx, publisher)

	return nil
}

// monitorDoneContext monitors when the publisher has finished and informs the observer when so
func (s *QUICPubServer) monitorDoneContext(ctx context.Context, publisher *QUICPublisherConn) {
	func() {
		for {
			select {
			case <-ctx.Done():
				if publisher != nil {
					s.observer.OnPublisherDisconnected(publisher)
				}
				return
			}
		}
	}()
}

func (s *QUICPubServer) receiveMessages(ctx context.Context, stream quic.ReceiveStream) error {
	// Don't timeout, as the publisher can stay idle until it starts sending messages
	if err := stream.SetReadDeadline(time.Time{}); err != nil {
		return errors.Wrap(err, "SetReadDeadline")
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping receiving messages as context is done")
			return nil
		default:
			if err := s.receiveMessage(stream); err != nil {
				return errors.Wrap(err, "receiveMessage")
			}
		}
	}
}

func (s *QUICPubServer) receiveMessage(stream quic.ReceiveStream) error {
	msg := make([]byte, s.config.MaxMessageBytes)
	n, err := stream.Read(msg)
	if err != nil {
		return errors.Wrap(err, "stream.Read")
	}
	msg = msg[0:n] // Trim to the number of bytes it read

	err = s.observer.OnPublisherMessage(msg)
	if err != nil {
		return errors.Wrap(err, "OnPublisherMessage")
	}

	return nil
}
