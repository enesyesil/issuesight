// Package auth provides authentication functionality for the Gateway service.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// StateKeyPrefix is the Redis key prefix for OAuth states.
	stateKeyPrefix = "oauth:state:"
	// StateTTL is the default TTL for OAuth states (10 minutes).
	stateTTL = 10 * time.Minute
)

// StateStore defines the interface for OAuth state storage.
type StateStore interface {
	// Save stores an OAuth state and returns the key (nonce).
	Save(ctx context.Context, state *OAuthState) error
	// Get retrieves an OAuth state by nonce and deletes it (one-time use).
	Get(ctx context.Context, nonce string) (*OAuthState, error)
	// Delete removes an OAuth state.
	Delete(ctx context.Context, nonce string) error
}

// MemoryStateStore stores OAuth states in memory.
// WARNING: Not suitable for production - states are lost on restart and not shared across instances.
type MemoryStateStore struct {
	states map[string]*OAuthState
}

// NewMemoryStateStore creates a new in-memory state store.
func NewMemoryStateStore() *MemoryStateStore {
	return &MemoryStateStore{
		states: make(map[string]*OAuthState),
	}
}

// Save stores an OAuth state in memory.
func (s *MemoryStateStore) Save(_ context.Context, state *OAuthState) error {
	s.states[state.Nonce] = state
	return nil
}

// Get retrieves and deletes an OAuth state from memory.
func (s *MemoryStateStore) Get(_ context.Context, nonce string) (*OAuthState, error) {
	state, ok := s.states[nonce]
	if !ok {
		return nil, fmt.Errorf("oauth state not found: %s", nonce)
	}
	delete(s.states, nonce) // One-time use
	return state, nil
}

// Delete removes an OAuth state from memory.
func (s *MemoryStateStore) Delete(_ context.Context, nonce string) error {
	delete(s.states, nonce)
	return nil
}

// RedisStateStore stores OAuth states in Redis.
// This is the recommended implementation for production.
type RedisStateStore struct {
	client *redis.Client
}

// NewRedisStateStore creates a new Redis-backed state store.
func NewRedisStateStore(client *redis.Client) (*RedisStateStore, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is required")
	}
	return &RedisStateStore{client: client}, nil
}

// Save stores an OAuth state in Redis with a TTL.
func (s *RedisStateStore) Save(ctx context.Context, state *OAuthState) error {
	key := stateKeyPrefix + state.Nonce

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal oauth state: %w", err)
	}

	// Set with TTL
	if err := s.client.Set(ctx, key, data, stateTTL).Err(); err != nil {
		return fmt.Errorf("save oauth state to redis: %w", err)
	}

	return nil
}

// Get retrieves and deletes an OAuth state from Redis (one-time use).
func (s *RedisStateStore) Get(ctx context.Context, nonce string) (*OAuthState, error) {
	key := stateKeyPrefix + nonce

	// Get the value
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("oauth state not found or expired: %s", nonce)
		}
		return nil, fmt.Errorf("get oauth state from redis: %w", err)
	}

	// Delete immediately (one-time use)
	s.client.Del(ctx, key)

	var state OAuthState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal oauth state: %w", err)
	}

	return &state, nil
}

// Delete removes an OAuth state from Redis.
func (s *RedisStateStore) Delete(ctx context.Context, nonce string) error {
	key := stateKeyPrefix + nonce
	return s.client.Del(ctx, key).Err()
}
