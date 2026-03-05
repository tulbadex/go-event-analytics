package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/go-redis/redis/v8"
)

type Store struct {
	client *redis.Client
	ttl    time.Duration
}

func NewStore(client *redis.Client, ttl time.Duration) *Store {
	return &Store{client: client, ttl: ttl}
}

func (s *Store) Create(ctx context.Context, userID string) (string, error) {
	token := generateToken()
	key := "session:" + token
	err := s.client.Set(ctx, key, userID, s.ttl).Err()
	return token, err
}

func (s *Store) Get(ctx context.Context, token string) (string, error) {
	key := "session:" + token
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	s.client.Expire(ctx, key, s.ttl)
	return val, nil
}

func (s *Store) Delete(ctx context.Context, token string) error {
	key := "session:" + token
	return s.client.Del(ctx, key).Err()
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
