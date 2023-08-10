package mocks

import (
	"context"
	"github.com/quic-go/quic-go"
	"sync/atomic"
	"time"
)

// MockStreamForSendingPings implements quick.Stream for testing pinging.
type MockStreamForSendingPings struct {
	TimesReadCalled  uint32
	TimesWriteCalled uint32
}

func (m *MockStreamForSendingPings) StreamID() quic.StreamID {
	return 123
}

func (m *MockStreamForSendingPings) Read([]byte) (n int, err error) {
	atomic.AddUint32(&m.TimesReadCalled, 1)
	return 1, nil
}

func (m *MockStreamForSendingPings) CancelRead(quic.StreamErrorCode) {
	panic("CancelRead not implemented")
}

func (m *MockStreamForSendingPings) SetReadDeadline(time.Time) error {
	return nil
}

func (m *MockStreamForSendingPings) Write([]byte) (n int, err error) {
	atomic.AddUint32(&m.TimesWriteCalled, 1)
	return 1, nil
}

func (m *MockStreamForSendingPings) Close() error {
	panic("Close not implemented")
}

func (m *MockStreamForSendingPings) CancelWrite(quic.StreamErrorCode) {
	panic("CancelWrite not implemented")
}

func (m *MockStreamForSendingPings) Context() context.Context {
	panic("Context not implemented")
}

func (m *MockStreamForSendingPings) SetWriteDeadline(time.Time) error {
	return nil
}

func (m *MockStreamForSendingPings) SetDeadline(time.Time) error {
	return nil
}

// MockStreamForReceivingPings implements quick.Stream for testing pinging.
type MockStreamForReceivingPings struct {
	SleepBeforeReads time.Duration // Simulate periodical writes to the ping stream by the peer
	TimesReadCalled  uint32
	TimesWriteCalled uint32
}

func (m *MockStreamForReceivingPings) StreamID() quic.StreamID {
	return 123
}

func (m *MockStreamForReceivingPings) Read([]byte) (n int, err error) {
	atomic.AddUint32(&m.TimesReadCalled, 1)
	time.Sleep(m.SleepBeforeReads)
	return 1, nil
}

func (m *MockStreamForReceivingPings) CancelRead(quic.StreamErrorCode) {
	panic("CancelRead not implemented")
}

func (m *MockStreamForReceivingPings) SetReadDeadline(time.Time) error {
	return nil
}

func (m *MockStreamForReceivingPings) Write([]byte) (n int, err error) {
	atomic.AddUint32(&m.TimesWriteCalled, 1)
	return 1, nil
}

func (m *MockStreamForReceivingPings) Close() error {
	panic("Close not implemented")
}

func (m *MockStreamForReceivingPings) CancelWrite(quic.StreamErrorCode) {
	panic("CancelWrite not implemented")
}

func (m *MockStreamForReceivingPings) Context() context.Context {
	panic("Context not implemented")
}

func (m *MockStreamForReceivingPings) SetWriteDeadline(time.Time) error {
	return nil
}

func (m *MockStreamForReceivingPings) SetDeadline(time.Time) error {
	return nil
}
