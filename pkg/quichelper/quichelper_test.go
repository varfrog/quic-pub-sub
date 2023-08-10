package quichelper_test

import (
	"bytes"
	"encoding/json"
	"errors"
	. "github.com/onsi/gomega"
	"github.com/varfrog/quicpubsub/pkg/quichelper"
	"github.com/varfrog/quicpubsub/pkg/sdk"
	"testing"
)

func TestReceiveEvent(t *testing.T) {
	t.Run("Valid event", func(t *testing.T) {
		g := NewWithT(t)

		msg, _ := json.Marshal(map[string]interface{}{
			"code": sdk.CodeNoSubscribers,
		})

		stream := bytes.NewReader(msg)

		event, err := quichelper.ReceiveEvent(stream, uint64(len(msg)))

		// Assertions
		g.Expect(err).To(BeNil())
		g.Expect(event).ToNot(BeNil())
		g.Expect(event.Code).To(Equal(sdk.CodeNoSubscribers))
	})

	t.Run("Invalid event", func(t *testing.T) {
		g := NewWithT(t)

		invalidMsg := []byte("{invalid_json}")

		stream := bytes.NewReader(invalidMsg)

		_, err := quichelper.ReceiveEvent(stream, uint64(len(invalidMsg)))

		// Assertions
		g.Expect(err).To(HaveOccurred())
		g.Expect(err).To(BeAssignableToTypeOf(&quichelper.UnmarshalError{}))
		var unmarshalError quichelper.UnmarshalError
		if errors.As(err, &unmarshalError) {
			g.Expect(unmarshalError.Data).To(Equal(invalidMsg))
		}
	})
}
