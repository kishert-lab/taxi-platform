package ws

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

func TestPassengerDriverLocationPayloadJSON(t *testing.T) {
	payload := PassengerDriverLocationPayload{
		OrderID:  uuid.New(),
		DriverID: uuid.New(),
		Status:   domain.OrderStatusDriverArriving,
		Location: dto.CoordinatesResponse{
			Latitude:  56.8,
			Longitude: 60.6,
		},
		RecordedAt: time.Now().UTC(),
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal passenger location payload: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(bytes, &decoded); err != nil {
		t.Fatalf("unmarshal passenger location payload: %v", err)
	}
	for _, key := range []string{"order_id", "driver_id", "status", "location", "recorded_at"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("expected payload key %s", key)
		}
	}
}
