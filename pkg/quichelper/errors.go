package quichelper

import (
	"errors"
	"fmt"
)

// ErrNetworkTimeout is a network timeout. Example usage is when we re-return net.Error when Timeout()=true
var ErrNetworkTimeout = errors.New("network timeout")

type UnmarshalError struct {
	Data []byte // data which failed failed to unmarshal
	Err  error
}

func (e UnmarshalError) Error() string {
	return fmt.Sprintf("unmarshall: %v", e.Err.Error())
}
