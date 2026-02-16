package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"knowledge-srv/internal/embedding/repository"
	"time"
)

const Prefix = "embedding:"

func (r *implRepository) Get(ctx context.Context, opt repository.GetOptions) ([]float32, error) {
	key := fmt.Sprintf("%s%s", Prefix, opt.Key)
	data, err := r.redis.GetClient().Get(ctx, key).Result()
	if err != nil {
		r.l.Errorf(ctx, "embedding.repository.redis.Get: %v", err)
		return nil, err
	}

	var vector []float32
	if err := json.Unmarshal([]byte(data), &vector); err != nil {
		r.l.Errorf(ctx, "embedding.repository.redis.Get: unmarshal error: %v", err)
		return nil, err
	}
	return vector, nil
}

func (r *implRepository) Save(ctx context.Context, opt repository.SaveOptions) error {
	key := fmt.Sprintf("%s%s", Prefix, opt.Key)
	data, err := json.Marshal(opt.Vector)
	if err != nil {
		r.l.Errorf(ctx, "embedding.repository.redis.Save: %v", err)
		return err
	}

	// Use TTL from options if provided, otherwise default to 7 days
	ttl := opt.TTL
	if ttl == 0 {
		ttl = 7 * 24 * time.Hour
	}

	if err := r.redis.GetClient().Set(ctx, key, data, ttl).Err(); err != nil {
		r.l.Errorf(ctx, "embedding.repository.redis.Save: %v", err)
		return err
	}
	return nil
}
