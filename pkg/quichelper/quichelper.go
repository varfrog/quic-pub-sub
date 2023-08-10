// Package quichelper contains helpers for working with QUIC and functions for deduplication of
// QUIC-specific code in apps.
package quichelper

import (
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/varfrog/quicpubsub/pkg/sdk"
	"io"
	"net"
)

// ReceiveEvent reads up to maxMessageBytes from the given stream and unmarshalls the result into sdk.Event.
// Returns UnmarshalError if the event is corrupt.
// Returns ErrNetworkTimeout on timeout.
func ReceiveEvent(stream io.Reader, maxMessageBytes uint64) (sdk.Event, error) {
	msg := make([]byte, maxMessageBytes)
	n, err := stream.Read(msg)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return sdk.Event{}, ErrNetworkTimeout
		}
		return sdk.Event{}, errors.Wrap(err, "stream.Read")
	}
	msg = msg[0:n] // Trim to the number of bytes it read

	var event sdk.Event
	if err := json.Unmarshal(msg, &event); err != nil {
		return sdk.Event{}, &UnmarshalError{Data: msg, Err: err}
	}

	return event, nil
}
