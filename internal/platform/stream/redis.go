// Package stream - Redis Streams implementation.
//
// This file contains the actual Redis implementation of the Publisher and
// Consumer interfaces defined in stream.go.
//
// REDIS STREAMS QUICK REFERENCE:
//   - XADD:       Add a message to a stream
//   - XREADGROUP: Read messages as part of a consumer group
//   - XACK:       Acknowledge a message was processed
//   - XGROUP:     Manage consumer groups
package stream

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// =============================================================================
// PUBLISHER IMPLEMENTATION
// =============================================================================

// RedisPublisher implements the Publisher interface using Redis Streams.
//
// It's a thin wrapper around the Redis client that knows how to format
// messages for the XADD command.
type RedisPublisher struct {
	// client is the Redis connection we'll use to send commands.
	// This is injected via the constructor (dependency injection pattern).
	client *redis.Client

	// config holds publisher settings like MaxLen
	config PublisherConfig
}

// NewRedisPublisher creates a new publisher that sends messages to Redis Streams.
//
// VALIDATION:
// Returns ErrNilClient if client is nil.
//
// EXAMPLE:
//
//	redisClient, _ := redis.NewClient(config)
//	publisher, err := stream.NewRedisPublisher(redisClient, stream.DefaultPublisherConfig())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	publisher.Publish(ctx, "my-stream", map[string]interface{}{"hello": "world"})
//
// WITH MAXLEN (recommended for production):
//
//	publisher, _ := stream.NewRedisPublisher(redisClient, stream.PublisherConfig{
//	    MaxLen: 10000,  // Keep only last 10k messages
//	})
func NewRedisPublisher(client *redis.Client, config PublisherConfig) (*RedisPublisher, error) {
	if client == nil {
		return nil, ErrNilClient
	}
	return &RedisPublisher{
		client: client,
		config: config,
	}, nil
}

// Publish adds a message to a Redis stream using the XADD command.
//
// WHAT HAPPENS UNDER THE HOOD:
//  1. Validates inputs (stream name and payload)
//  2. Converts your map into the format Redis expects
//  3. Sends: XADD stream-name * key1 value1 key2 value2 ...
//  4. If MaxLen is set, trims old messages
//  5. Redis stores the message and returns a unique ID
//  6. We return that ID to you
//
// The "*" tells Redis to auto-generate the message ID using the current
// timestamp. This ensures messages are ordered chronologically.
//
// VALIDATION:
//   - stream cannot be empty (returns ErrEmptyStreamName)
//   - payload cannot be empty (returns ErrEmptyPayload)
func (p *RedisPublisher) Publish(ctx context.Context, stream string, payload map[string]interface{}) (string, error) {
	// Validate inputs
	if stream == "" {
		return "", ErrEmptyStreamName
	}
	if len(payload) == 0 {
		return "", ErrEmptyPayload
	}

	// Redis XADD expects values as a flat list: [key1, val1, key2, val2, ...]
	// So we need to convert our map into this format.
	//
	// WHY len(payload)*2?
	// For each key-value pair, we need 2 slots: one for the key, one for the value.
	values := make([]interface{}, 0, len(payload)*2)
	for k, v := range payload {
		values = append(values, k, v)
	}

	// Build the XADD arguments
	args := &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}

	// Add MaxLen if configured (for stream trimming)
	// Using MaxLen with ~ (approximate) for better performance
	if p.config.MaxLen > 0 {
		args.MaxLen = p.config.MaxLen
		args.Approx = true // Use ~ for approximate trimming (more efficient)
	}

	// XADD is the Redis command to add a message to a stream.
	// We're not setting an ID, so Redis will auto-generate one like "1609459200000-0"
	result, err := p.client.XAdd(ctx, args).Result()
	if err != nil {
		return "", fmt.Errorf("failed to publish to stream %s: %w", stream, err)
	}

	// result is the message ID that Redis assigned, e.g., "1609459200000-0"
	return result, nil
}

// =============================================================================
// CONSUMER IMPLEMENTATION
// =============================================================================

// RedisConsumer implements the Consumer interface using Redis Streams.
//
// It manages a consumer group, which allows multiple consumers to share
// the work of processing messages from a stream.
type RedisConsumer struct {
	// client is the Redis connection for sending commands
	client *redis.Client

	// config controls how we consume (batch size, block duration, etc.)
	config ConsumerConfig
}

// NewRedisConsumer creates a new consumer for reading from Redis Streams.
//
// VALIDATION:
// Returns ErrNilClient if client is nil.
//
// EXAMPLE:
//
//	consumer, err := stream.NewRedisConsumer(redisClient, stream.DefaultConsumerConfig())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	consumer.CreateGroup(ctx, "github-events", "ai-workers")
//	consumer.Consume(ctx, "github-events", "ai-workers", "worker-1", handleMessage)
func NewRedisConsumer(client *redis.Client, config ConsumerConfig) (*RedisConsumer, error) {
	if client == nil {
		return nil, ErrNilClient
	}
	return &RedisConsumer{
		client: client,
		config: config,
	}, nil
}

// CreateGroup creates a consumer group for a stream.
//
// WHAT IS A CONSUMER GROUP?
// It's a way to have multiple consumers share the work:
//   - Each message goes to exactly ONE consumer in the group
//   - Redis tracks which messages are pending (delivered but not acknowledged)
//   - If a consumer crashes, pending messages can be claimed by others
//
// THE "0" ARGUMENT:
// This tells Redis where to start reading from:
//   - "0"    = Start from the beginning of the stream
//   - "$"    = Start from new messages only (ignore existing)
//   - "<ID>" = Start from a specific message ID
//
// MKSTREAM:
// The MKSTREAM option creates the stream if it doesn't exist.
// This is convenient because you don't have to create the stream separately.
//
// VALIDATION:
//   - stream cannot be empty (returns ErrEmptyStreamName)
//   - group cannot be empty (returns ErrEmptyGroupName)
func (c *RedisConsumer) CreateGroup(ctx context.Context, stream, group string) error {
	// Validate inputs
	if stream == "" {
		return ErrEmptyStreamName
	}
	if group == "" {
		return ErrEmptyGroupName
	}

	// XGROUP CREATE stream-name group-name 0 MKSTREAM
	//
	// This command creates a consumer group. If it already exists, Redis
	// returns an error with "BUSYGROUP" - we ignore that because it's fine.
	err := c.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()

	// Ignore the "already exists" error - it's not really an error for us
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group %s for stream %s: %w", group, stream, err)
	}
	return nil
}

// Consume starts an infinite loop that processes messages from a stream.
//
// THE CONSUME LOOP:
//  1. Ask Redis for new messages (blocks up to BlockDuration)
//  2. For each message received:
//     a. Call your handler function
//     b. If handler succeeds, acknowledge the message
//  3. Go back to step 1
//
// STOPPING THE LOOP:
// The loop continues until ctx is cancelled. In a real app:
//   - You'd create a context with cancel: ctx, cancel := context.WithCancel(...)
//   - When shutting down (e.g., SIGTERM), call cancel()
//   - The loop will exit gracefully
//
// ERROR HANDLING:
// If your handler returns an error, we stop the loop and return that error.
// In production, you might want more sophisticated error handling (retries, DLQ, etc.)
//
// VALIDATION:
//   - stream cannot be empty (returns ErrEmptyStreamName)
//   - group cannot be empty (returns ErrEmptyGroupName)
//   - consumer cannot be empty (returns ErrEmptyConsumerName)
//   - handler cannot be nil (returns ErrNilHandler)
func (c *RedisConsumer) Consume(ctx context.Context, stream, group, consumer string, handler func(Message) error) error {
	// Validate inputs upfront (fail fast)
	if stream == "" {
		return ErrEmptyStreamName
	}
	if group == "" {
		return ErrEmptyGroupName
	}
	if consumer == "" {
		return ErrEmptyConsumerName
	}
	if handler == nil {
		return ErrNilHandler
	}

	// This is our main consume loop - it runs forever until cancelled
	for {
		// First, check if we should stop (context was cancelled)
		// This is the standard Go pattern for checking cancellation
		select {
		case <-ctx.Done():
			// Context was cancelled - time to shut down
			return ctx.Err()
		default:
			// Not cancelled, continue processing
		}

		// XREADGROUP reads messages from a stream as part of a consumer group.
		//
		// The ">" is special - it means "give me new messages that haven't
		// been delivered to any consumer in this group yet."
		//
		// Other options:
		//   - "0" = Give me my pending messages (ones I received but didn't ack)
		//   - "<ID>" = Give me messages after this specific ID
		//
		// Block: How long to wait if there are no messages (0 = wait forever)
		// Count: Max number of messages to return per call
		streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Consumer: consumer,
			Streams:  []string{stream, ">"},
			Count:    c.config.Count,
			Block:    c.config.BlockDuration,
		}).Result()

		if err != nil {
			// redis.Nil means the blocking call timed out with no messages.
			// This is totally normal - just loop and try again.
			if err == redis.Nil {
				continue
			}
			// Any other error is a real problem
			return fmt.Errorf("failed to read from stream %s: %w", stream, err)
		}

		// Process each message we received
		// 'streams' is a list because you CAN read from multiple streams at once
		// (we only read from one, but the API supports multiple)
		for _, streamData := range streams {
			for _, msg := range streamData.Messages {
				// Convert Redis's message format into our Message struct
				message := Message{
					ID:      msg.ID,
					Stream:  streamData.Stream,
					Payload: msg.Values,
				}

				// Call the user's handler function to process the message
				// This is where your business logic goes!
				if err := handler(message); err != nil {
					// Handler failed - we return the error and stop processing.
					// The message is NOT acknowledged, so it can be retried.
					//
					// In a production system, you might want to:
					//   - Log the error and continue
					//   - Move to a "dead letter queue" after N retries
					//   - etc.
					return fmt.Errorf("handler error for message %s: %w", msg.ID, err)
				}

				// Handler succeeded! Acknowledge the message so Redis knows
				// we're done with it and doesn't redeliver it.
				if err := c.Ack(ctx, stream, group, msg.ID); err != nil {
					return fmt.Errorf("failed to ack message %s: %w", msg.ID, err)
				}
			}
		}
	}
}

// Ack acknowledges that a message was successfully processed.
//
// WHY ACKNOWLEDGE?
// When you receive a message via XREADGROUP, Redis marks it as "pending" -
// meaning "Consumer X has this, waiting for confirmation."
//
// When you XACK the message, you're saying "I'm done processing this."
// Redis then removes it from the pending list.
//
// WHAT IF YOU DON'T ACK?
// The message stays in the pending list. If your consumer crashes,
// another consumer can "claim" the message and retry it.
// This is how Redis Streams provides at-least-once delivery.
//
// VALIDATION:
//   - stream cannot be empty (returns ErrEmptyStreamName)
//   - group cannot be empty (returns ErrEmptyGroupName)
//   - messageID cannot be empty (returns error)
func (c *RedisConsumer) Ack(ctx context.Context, stream, group, messageID string) error {
	// Validate inputs
	if stream == "" {
		return ErrEmptyStreamName
	}
	if group == "" {
		return ErrEmptyGroupName
	}
	if messageID == "" {
		return errors.New("stream: message ID cannot be empty")
	}

	// XACK stream-name group-name message-id
	// Tells Redis "I'm done with this message, you can forget about it"
	if err := c.client.XAck(ctx, stream, group, messageID).Err(); err != nil {
		return fmt.Errorf("failed to ack message %s: %w", messageID, err)
	}
	return nil
}
