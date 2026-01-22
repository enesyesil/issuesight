// Package stream provides message streaming capabilities using Redis Streams.


// WHAT IS A MESSAGE STREAM?
// Imagine a conveyor belt in a factory:
//   - Producers put items (messages) onto the belt
//   - Consumers pick items off the belt and process them
//   - Items stay on the belt until someone picks them up
//   - Multiple workers can share the work (each item goes to only one worker)

// WHY USE STREAMS?
// In IssueSight, we use streams to pass work between services:
//   1. The Collector service fetches GitHub issues
//   2. It puts each issue onto the "github-events" stream
//   3. The AI Processor service picks up issues from the stream
//   4. The AI Processor generates tutorials for each issue

// This "decoupling" is powerful because:
//   - If the AI Processor is slow, issues queue up instead of being lost
//   - We can run multiple AI Processors to handle more work
//   - If one service crashes, the other keeps running

// REDIS STREAMS vs OTHER OPTIONS:
//   - Simpler than Kafka/RabbitMQ (no extra infrastructure)
//   - Built into Redis (we're already using it for caching)
//   - Supports "consumer groups" for distributing work

package stream

import (
	"context"
	"errors"
	"time"
)



// These errors are returned for invalid inputs.
// Using typed errors lets you check specifically what went wrong:


var (
	// ErrNilClient is returned when a nil Redis client is provided.
	ErrNilClient = errors.New("stream: redis client cannot be nil")

	// ErrEmptyStreamName is returned when stream name is empty.
	ErrEmptyStreamName = errors.New("stream: stream name cannot be empty")

	// ErrEmptyGroupName is returned when consumer group name is empty.
	ErrEmptyGroupName = errors.New("stream: group name cannot be empty")

	// ErrEmptyConsumerName is returned when consumer name is empty.
	ErrEmptyConsumerName = errors.New("stream: consumer name cannot be empty")

	// ErrNilHandler is returned when a nil handler function is provided.
	ErrNilHandler = errors.New("stream: handler function cannot be nil")

	// ErrEmptyPayload is returned when trying to publish an empty payload.
	ErrEmptyPayload = errors.New("stream: payload cannot be empty")
)



// Message represents a single message from the stream.

// When you consume from a stream, you get Message objects containing:
//   - ID:      A unique identifier 
//   - Stream:  Which stream this came from
//   - Payload: The actual data as key-value pairs


type Message struct {
	// ID is a unique message identifier assigned by Redis.
	// Format: "<timestamp>-<sequence>"
	// The timestamp is when the message was added (milliseconds since epoch).
	// The sequence handles multiple messages at the same millisecond.
	ID string

	// Stream is the name of the stream this message came from.

	// Useful when consuming from multiple streams.
	Stream string

	// Payload contains the message data as key-value pairs. You decide what keys to use based on your application needs.
	// Values can be strings, numbers, etc.

	Payload map[string]interface{}
}



// Publisher is the interface for sending messages to a stream.

// Think of it like a person putting packages onto a conveyor belt.  You just need to specify which belt (stream) and what's in the package (payload).
type Publisher interface {
	// Publish adds a message to the specified stream.
	
	// Parameters:
	//   - ctx:     Context for cancellation/timeout
	//   - stream:  Name of the stream ("github-events")
	//   - payload: The message data as key-value pairs (cannot be empty)
	
	// Returns:
	//   - string: The message ID assigned by Redis 
	//   - error:  Any error that occurred
	
	// VALIDATION:
	//   - stream cannot be empty (returns ErrEmptyStreamName)
	//   - payload cannot be empty (returns ErrEmptyPayload)
	
	
	Publish(ctx context.Context, stream string, payload map[string]interface{}) (string, error)
}



// Consumer is the interface for receiving messages from a stream.



// TERMINOLOGY:
//   - Stream:   The conveyor belt ( "github-events")
//   - Group:    A team of workers ( "ai-workers")
//   - Consumer: One worker in the team ( "worker-1", "worker-2")
type Consumer interface {
	// Consume starts an infinite loop that processes messages from a stream.
	

	
	// Parameters:
	//   - ctx:      Context for cancellation (when shutting down)
	//   - stream:   Name of the stream to consume from
	//   - group:    Consumer group name (for coordinating multiple consumers)
	//   - consumer: This consumer's unique name within the group
	//   - handler:  Your function that processes each message
	


	Consume(ctx context.Context, stream, group, consumer string, handler func(Message) error) error

	// CreateGroup creates a consumer group for a stream.
	
	// You MUST call this before consuming! It tells Redis:
	//   - "Create a group called X for stream Y"
	//   - "Start reading from the beginning of the stream"
	
	// Safe to call multiple times - if the group exists, it does nothing.
	
	CreateGroup(ctx context.Context, stream, group string) error

	// Ack acknowledges that a message was successfully processed.
	
	// WHY ACKNOWLEDGE?
	// Redis tracks which messages are "pending" (delivered but not done).
	// When you ack a message, you're saying "I'm done with this, delete it."
	// If you crash before acking, Redis can redeliver to another consumer.

	Ack(ctx context.Context, stream, group, messageID string) error
}



// PublisherConfig holds settings for the publisher.
type PublisherConfig struct {
	// MaxLen is the maximum number of messages to keep in the stream.
	
	// WHY LIMIT STREAM SIZE?
	// Without a limit, streams grow forever and consume memory.
	// MaxLen automatically trims old messages when the limit is exceeded.
	
	// HOW TRIMMING WORKS: 
	// When you publish and the stream exceeds MaxLen, Redis removes the oldest messages. This is approximate (for performance) - the actual count may be slightly higher.
	
	// CHOOSING A VALUE:
	//   - 0:      No limit (stream grows forever) - use for small volumes
	//   - 10000:  Keep last 10k messages - good for most use cases
	//   - 100000: Keep last 100k messages - for high-volume streams
	
	// Default (0): No limit.
	MaxLen int64
}

// DefaultPublisherConfig returns sensible defaults for publishers.
func DefaultPublisherConfig() PublisherConfig {
	return PublisherConfig{
		MaxLen: 0, // No limit by default - set based on your needs
	}
}

// ConsumerConfig holds settings that control how the consumer behaves.
type ConsumerConfig struct {
	// BlockDuration is how long to wait for new messages before checking again.
	
	// WHY BLOCK?
	// Instead of constantly asking "any new messages? any new messages?",
	// we tell Redis "wait up to 5 seconds, then let me know."
	// This is more efficient than polling in a tight loop.
	BlockDuration time.Duration

	// Count is how many messages to fetch in each batch.
	
	// TRADEOFF:
	//   - Higher count = fewer round trips to Redis = more efficient
	//   - Lower count  = faster response to individual messages
	// 10 is a reasonable default for most use cases.

	Count int64
}

// DefaultConsumerConfig returns sensible default settings for most use cases.

// You can use this as a starting point and adjust if needed:
//   - BlockDuration: 5 seconds (good balance of responsiveness vs efficiency)
//   - Count: 10 messages per batch

func DefaultConsumerConfig() ConsumerConfig {
	return ConsumerConfig{
		BlockDuration: 5 * time.Second,
		Count:         10,
	}
}
