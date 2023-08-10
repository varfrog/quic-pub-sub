package quichelper

import (
	"context"
	"github.com/pkg/errors"
	"github.com/quic-go/quic-go"
	"go.uber.org/zap"
	"net"
	"time"
)

type PingerConfig struct {
	Interval time.Duration // Repeat pings at this interval
	Timeout  time.Duration // Ping request read/write deadline exceeding which the peer is considered dead
}

// Pinger is used to accept or send continuous ping requests to another peer.
type Pinger struct {
	config PingerConfig
	logger *zap.Logger
}

func NewDefaultPingerConfig() PingerConfig {
	return PingerConfig{
		Interval: time.Second * 5,
		Timeout:  time.Second * 10,
	}
}

// NewPinger is the constructor for Pinger.
func NewPinger(config PingerConfig, logger *zap.Logger) *Pinger {
	return &Pinger{config: config, logger: logger}
}

// SendPings continuously writes ping requests to the given stream at an interval, and reads responses.
// It serves both as a keep-alive request and also so that the peer knows the other peer is gone.
// Returns quichelper.ErrNetworkTimeout when ending takes longer than the timeout or when no incoming ping comes
// in for longer than the timeout.
func (s *Pinger) SendPings(ctx context.Context, stream quic.Stream) error {
	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("Stopping sending pings, context cancelled")
			return nil
		case <-time.After(s.config.Interval):
			if err := s.sendPing(stream); err != nil {
				return errors.Wrap(err, "sendPing")
			}
			s.logger.Debug("Sent ping")

			if err := s.waitForPing(stream); err != nil {
				return errors.Wrap(err, "waitForPing")
			}
			s.logger.Debug("Got response to ping")
		}
	}
}

// AcceptPings accepts ping requests from a peer and to each responds with a its own ping request.
// Returns ErrNetworkTimeout when no requests arrive within the timeout or if sending exceeds the timeout.
func (s *Pinger) AcceptPings(ctx context.Context, stream quic.Stream) error {
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping accepting pings, context cancelled")
			return nil
		default:
			// Accept a ping message
			if err := s.waitForPing(stream); err != nil {
				return errors.Wrap(err, "waitForPing")
			}
			s.logger.Debug("Got ping")

			// Respond
			if err := s.sendPing(stream); err != nil {
				return errors.Wrap(err, "sendPing")
			}
			s.logger.Debug("Responded to ping")
		}
	}
}

// sendPing sends a ping request on the given stream with a timeout.
// It returns ErrNetworkTimeout on time out.
func (s *Pinger) sendPing(stream quic.Stream) error {
	if err := stream.SetWriteDeadline(time.Now().Add(s.config.Timeout)); err != nil {
		return errors.Wrap(err, "SetWriteDeadline")
	}

	_, err := stream.Write([]byte{'1'})
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return ErrNetworkTimeout
		} else {
			return errors.Wrap(err, "stream.Write")
		}
	}
	return nil
}

// waitForPing waits on stream for a ping response with a timeout.
// Returns ErrNetworkTimeout on timeout.
func (s *Pinger) waitForPing(stream quic.Stream) error {
	if err := stream.SetReadDeadline(time.Now().Add(s.config.Timeout)); err != nil {
		return errors.Wrap(err, "SetReadDeadline")
	}

	_, err := stream.Read(make([]byte, 1))
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return ErrNetworkTimeout
		} else {
			return errors.Wrap(err, "stream.Read")
		}
	}
	return nil
}
