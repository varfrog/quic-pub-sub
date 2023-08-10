package quichelper_test

import (
	"context"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/varfrog/quicpubsub/pkg/quichelper"
	"github.com/varfrog/quicpubsub/pkg/quichelper/mocks"
	"go.uber.org/zap"
	"testing"
	"time"
)

func TestPinger(t *testing.T) {
	pingInterval := time.Millisecond * 50
	pinger := quichelper.NewPinger(quichelper.PingerConfig{
		Interval: pingInterval,
		Timeout:  time.Second, // Not important here
	}, zap.NewNop())

	t.Run("Sends a number of pings and responds to them", func(t *testing.T) {
		g := NewWithT(t)

		// Continuously ping for some time
		ctx, cancel := context.WithTimeout(context.Background(), pingInterval*6)
		defer cancel()

		stream := &mocks.MockStreamForSendingPings{}
		g.Expect(pinger.SendPings(ctx, stream)).To(Succeed())

		assert.GreaterOrEqual(t, stream.TimesWriteCalled, uint32(5))
		assert.GreaterOrEqual(t, stream.TimesReadCalled, uint32(5))
	})

	t.Run("Accepts incoming pings and responds to them", func(t *testing.T) {
		g := NewWithT(t)

		// Continuously accept pings for some time
		ctx, cancel := context.WithTimeout(context.Background(), pingInterval*6)
		defer cancel()

		stream := &mocks.MockStreamForReceivingPings{
			SleepBeforeReads: pingInterval,
		}
		go func() {
			g.Expect(pinger.AcceptPings(ctx, stream)).To(Succeed())
		}()

		// Wait for some time for this test to finish sending pings and the Pinger to accept pings
		time.Sleep(stream.SleepBeforeReads * 6)

		assert.GreaterOrEqual(t, stream.TimesWriteCalled, uint32(5))
		assert.GreaterOrEqual(t, stream.TimesReadCalled, uint32(5))
	})
}
