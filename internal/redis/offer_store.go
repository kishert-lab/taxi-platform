package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/develoop/taxi-platform/internal/dispatch"
)

type OfferStore struct {
	client *goredis.Client
}

func NewOfferStore(client *goredis.Client) *OfferStore {
	return &OfferStore{client: client}
}

func (store *OfferStore) SaveOffer(ctx context.Context, offer dispatch.OrderOffer, ttl time.Duration) error {
	payload, err := json.Marshal(offer)
	if err != nil {
		return fmt.Errorf("marshal order offer: %w", err)
	}

	key := offerKey(offer.OrderID, offer.DriverID)
	if err := store.client.Set(ctx, key, payload, ttl).Err(); err != nil {
		return fmt.Errorf("save order offer: %w", err)
	}
	if err := store.client.SAdd(ctx, orderOffersKey(offer.OrderID), offer.DriverID.String()).Err(); err != nil {
		return fmt.Errorf("index order offer driver: %w", err)
	}
	if err := store.client.SAdd(ctx, driverOffersKey(offer.DriverID), offer.OrderID.String()).Err(); err != nil {
		return fmt.Errorf("index driver offer order: %w", err)
	}
	if err := store.client.Expire(ctx, orderOffersKey(offer.OrderID), ttl).Err(); err != nil {
		return fmt.Errorf("expire order offers index: %w", err)
	}
	if err := store.client.Expire(ctx, driverOffersKey(offer.DriverID), ttl).Err(); err != nil {
		return fmt.Errorf("expire driver offers index: %w", err)
	}

	return nil
}

func (store *OfferStore) GetOffer(ctx context.Context, orderID uuid.UUID, driverID uuid.UUID) (dispatch.OrderOffer, bool, error) {
	value, err := store.client.Get(ctx, offerKey(orderID, driverID)).Bytes()
	if err == goredis.Nil {
		return dispatch.OrderOffer{}, false, nil
	}
	if err != nil {
		return dispatch.OrderOffer{}, false, fmt.Errorf("get order offer: %w", err)
	}

	var offer dispatch.OrderOffer
	if err := json.Unmarshal(value, &offer); err != nil {
		return dispatch.OrderOffer{}, false, fmt.Errorf("unmarshal order offer: %w", err)
	}

	return offer, true, nil
}

func (store *OfferStore) ListOfferedDriverIDs(ctx context.Context, orderID uuid.UUID) ([]uuid.UUID, error) {
	values, err := store.client.SMembers(ctx, orderOffersKey(orderID)).Result()
	if err != nil {
		return nil, fmt.Errorf("list order offered drivers: %w", err)
	}

	driverIDs := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		driverID, err := uuid.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("parse offered driver id: %w", err)
		}
		driverIDs = append(driverIDs, driverID)
	}

	return driverIDs, nil
}

func (store *OfferStore) RemoveOffers(ctx context.Context, orderID uuid.UUID) error {
	driverIDs, err := store.ListOfferedDriverIDs(ctx, orderID)
	if err != nil {
		return err
	}

	keys := make([]string, 0, len(driverIDs)+1)
	for _, driverID := range driverIDs {
		keys = append(keys, offerKey(orderID, driverID))
		if err := store.client.SRem(ctx, driverOffersKey(driverID), orderID.String()).Err(); err != nil {
			return fmt.Errorf("remove order from driver offers index: %w", err)
		}
	}
	keys = append(keys, orderOffersKey(orderID))

	if err := store.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("remove order offers: %w", err)
	}

	return nil
}

func offerKey(orderID uuid.UUID, driverID uuid.UUID) string {
	return fmt.Sprintf("order:%s:driver:%s:offer", orderID, driverID)
}

func orderOffersKey(orderID uuid.UUID) string {
	return fmt.Sprintf("order:%s:offers", orderID)
}

func driverOffersKey(driverID uuid.UUID) string {
	return fmt.Sprintf("driver:%s:offers", driverID)
}
