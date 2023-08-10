package quichelper

import (
	"context"
	"errors"
	"github.com/quic-go/quic-go"
	"go.uber.org/zap"
)

// SendPings is a helper for code deduplication in the connectors to send ping requests,
// write logs and running cancel() on failure.
func SendPings(
	ctx context.Context,
	pinger *Pinger,
	streamCh <-chan quic.Stream,
	cancel func(),
	logger *zap.Logger,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case stream := <-streamCh:
			logger.Info("Starting to send pings")
			if err := pinger.SendPings(ctx, stream); err != nil {
				if errors.Is(err, ErrNetworkTimeout) {
					logger.Info("Got timeout from server when pinging, server possibly down, will shut down")
				} else {
					logger.Error("SendPings", zap.Error(err))
				}
				cancel()
				return
			}
		}
	}
}

// ReceiveAndRespondToPings is a helper for code deduplication in the server. It:
// 1. waits for a ping stream to become available
// 2. once the stream is available starts receiving and responding to ping results
// 3. calls cancel() on failure
func ReceiveAndRespondToPings(
	ctx context.Context,
	pinger *Pinger,
	streamCh <-chan quic.Stream,
	cancel func(),
	logMessageOnTimeout string,
	logger *zap.Logger,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case stream := <-streamCh:
			logger.Info("Accepting pings")
			if err := pinger.AcceptPings(ctx, stream); err != nil {
				if errors.Is(err, ErrNetworkTimeout) {
					logger.Info(logMessageOnTimeout)
				} else {
					logger.Error("AcceptPings", zap.Error(err))
				}
				cancel()
				return
			}
		}
	}
}
