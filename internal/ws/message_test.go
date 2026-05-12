package ws

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestMessageEnvelopeJSON(t *testing.T) {
	t.Parallel()

	message := NewMessage("order.driver_assigned", uuid.New(), map[string]any{"order_id": "order-1"})
	payload, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal ws message: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal ws message: %v", err)
	}
	for _, key := range []string{"event", "request_id", "occurred_at", "payload"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("expected envelope key %s", key)
		}
	}
}
