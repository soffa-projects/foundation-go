package adapters

import (
	"context"
	"sync"
	"testing"
	"time"

	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/test"
)

// ------------------------------------------------------------------------------------------------------------------
// Factory Functions Tests
// ------------------------------------------------------------------------------------------------------------------

func TestNewPubSubProvider_Fake(t *testing.T) {
	assert := test.NewAssertions(t)

	// Test all fake provider schemes
	providers := []string{"fake://provider", "faker://provider", "dummy://provider"}

	for _, provider := range providers {
		pubsub, err := NewPubSubProvider(provider)
		assert.Nil(err)
		assert.NotNil(pubsub)
		// Verify it implements the interface
		var _ f.PubSubProvider = pubsub
	}
}

func TestNewPubSubProvider_UnsupportedScheme(t *testing.T) {
	assert := test.NewAssertions(t)

	pubsub, err := NewPubSubProvider("unsupported://localhost")

	assert.NotNil(err)
	if pubsub != nil {
		t.Error("Expected nil pubsub for unsupported scheme")
	}
}

func TestNewPubSubProvider_InvalidURL(t *testing.T) {
	assert := test.NewAssertions(t)

	pubsub, err := NewPubSubProvider(":::invalid:::")

	assert.NotNil(err)
	if pubsub != nil {
		t.Error("Expected nil pubsub for invalid URL")
	}
}

func TestMustNewPubSubProvider_Success(t *testing.T) {
	assert := test.NewAssertions(t)

	// Should not panic with valid provider
	pubsub := MustNewPubSubProvider("fake://provider")

	assert.NotNil(pubsub)
}

func TestMustNewPubSubProvider_Panic(t *testing.T) {
	assert := test.NewAssertions(t)

	// Should panic with invalid provider
	defer func() {
		r := recover()
		assert.NotNil(r)
	}()

	MustNewPubSubProvider("invalid://provider")
	t.Error("Should have panicked")
}

// ------------------------------------------------------------------------------------------------------------------
// FakePubSubProvider Tests
// ------------------------------------------------------------------------------------------------------------------

func TestNewFakePubSubProvider(t *testing.T) {
	assert := test.NewAssertions(t)

	pubsub := NewFakePubSubProvider()

	assert.NotNil(pubsub)
	// Verify it implements the interface
	var _ f.PubSubProvider = pubsub
}

func TestFakePubSubProvider_Init(t *testing.T) {
	assert := test.NewAssertions(t)

	pubsub := NewFakePubSubProvider()
	err := pubsub.Init()

	assert.Nil(err)
}

func TestFakePubSubProvider_Ping(t *testing.T) {
	assert := test.NewAssertions(t)

	pubsub := NewFakePubSubProvider()
	err := pubsub.Ping()

	assert.Nil(err)
}

func TestFakePubSubProvider_PublishAndSubscribe(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	pubsub := NewFakePubSubProvider()

	// Set up a subscriber
	var received string
	var wg sync.WaitGroup
	wg.Add(1)

	pubsub.Subscribe(ctx, "test-topic", func(ctx context.Context, message string) {
		received = message
		wg.Done()
	})

	// Publish a message
	err := pubsub.Publish(ctx, "test-topic", "hello world")
	assert.Nil(err)

	// Wait for message to be received
	wg.Wait()
	assert.Equals(received, "hello world")
}

func TestFakePubSubProvider_SentCounter(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	pubsub := NewFakePubSubProvider().(*FakePubSubProvider)

	// Verify initial count is 0
	assert.Equals(pubsub.Sent("topic1"), 0)

	// Publish a message
	pubsub.Publish(ctx, "topic1", "message1")
	assert.Equals(pubsub.Sent("topic1"), 1)

	// Publish another message
	pubsub.Publish(ctx, "topic1", "message2")
	assert.Equals(pubsub.Sent("topic1"), 2)

	// Different topic should be independent
	assert.Equals(pubsub.Sent("topic2"), 0)
}

func TestFakePubSubProvider_ReceivedCounter(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	pubsub := NewFakePubSubProvider().(*FakePubSubProvider)

	// Subscribe to a topic
	pubsub.Subscribe(ctx, "topic1", func(ctx context.Context, message string) {
		// Handler does nothing
	})

	// Verify initial count is 0
	assert.Equals(pubsub.Received("topic1"), 0)

	// Publish a message
	pubsub.Publish(ctx, "topic1", "message1")
	// Give goroutine a moment to execute
	time.Sleep(10 * time.Millisecond)
	assert.Equals(pubsub.Received("topic1"), 1)

	// Publish another message
	pubsub.Publish(ctx, "topic1", "message2")
	time.Sleep(10 * time.Millisecond)
	assert.Equals(pubsub.Received("topic1"), 2)
}

func TestFakePubSubProvider_MultipleSubscribers(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	pubsub := NewFakePubSubProvider()

	// Set up multiple subscribers on the same topic
	var received1, received2 string
	var wg sync.WaitGroup
	wg.Add(2)

	pubsub.Subscribe(ctx, "topic", func(ctx context.Context, message string) {
		received1 = message
		wg.Done()
	})

	pubsub.Subscribe(ctx, "topic", func(ctx context.Context, message string) {
		received2 = message
		wg.Done()
	})

	// Publish a message
	err := pubsub.Publish(ctx, "topic", "broadcast")
	assert.Nil(err)

	// Wait for both subscribers to receive
	wg.Wait()
	assert.Equals(received1, "broadcast")
	assert.Equals(received2, "broadcast")
}

func TestFakePubSubProvider_MultipleTopics(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	pubsub := NewFakePubSubProvider()

	// Set up subscribers on different topics
	var received1, received2 string
	var wg sync.WaitGroup
	wg.Add(2)

	pubsub.Subscribe(ctx, "topic1", func(ctx context.Context, message string) {
		received1 = message
		wg.Done()
	})

	pubsub.Subscribe(ctx, "topic2", func(ctx context.Context, message string) {
		received2 = message
		wg.Done()
	})

	// Publish to different topics
	pubsub.Publish(ctx, "topic1", "message1")
	pubsub.Publish(ctx, "topic2", "message2")

	// Wait for both to receive
	wg.Wait()
	assert.Equals(received1, "message1")
	assert.Equals(received2, "message2")
}

func TestFakePubSubProvider_PublishWithoutSubscribers(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	pubsub := NewFakePubSubProvider().(*FakePubSubProvider)

	// Publish without any subscribers
	err := pubsub.Publish(ctx, "empty-topic", "message")
	assert.Nil(err)

	// Should still increment sent counter
	assert.Equals(pubsub.Sent("empty-topic"), 1)
	// Received should be 0 (no subscribers)
	assert.Equals(pubsub.Received("empty-topic"), 0)
}

func TestFakePubSubProvider_EmptyTopic(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	pubsub := NewFakePubSubProvider()

	// Subscribe and publish with empty topic
	var received string
	var wg sync.WaitGroup
	wg.Add(1)

	pubsub.Subscribe(ctx, "", func(ctx context.Context, message string) {
		received = message
		wg.Done()
	})

	err := pubsub.Publish(ctx, "", "empty topic message")
	assert.Nil(err)

	wg.Wait()
	assert.Equals(received, "empty topic message")
}

func TestFakePubSubProvider_EmptyMessage(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	pubsub := NewFakePubSubProvider()

	// Subscribe and publish empty message
	var received string
	var wg sync.WaitGroup
	wg.Add(1)

	pubsub.Subscribe(ctx, "topic", func(ctx context.Context, message string) {
		received = message
		wg.Done()
	})

	err := pubsub.Publish(ctx, "topic", "")
	assert.Nil(err)

	wg.Wait()
	assert.Equals(received, "")
}

func TestFakePubSubProvider_LargeMessage(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	pubsub := NewFakePubSubProvider()

	// Create a large message
	largeMessage := string(make([]byte, 10000))
	for i := range largeMessage {
		largeMessage = largeMessage[:i] + "a" + largeMessage[i+1:]
	}

	var received string
	var wg sync.WaitGroup
	wg.Add(1)

	pubsub.Subscribe(ctx, "topic", func(ctx context.Context, message string) {
		received = message
		wg.Done()
	})

	err := pubsub.Publish(ctx, "topic", largeMessage)
	assert.Nil(err)

	wg.Wait()
	assert.Equals(len(received), len(largeMessage))
}

func TestFakePubSubProvider_MultiplePublishes(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	pubsub := NewFakePubSubProvider()

	// Collect all received messages
	var messages []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(3)

	pubsub.Subscribe(ctx, "topic", func(ctx context.Context, message string) {
		mu.Lock()
		messages = append(messages, message)
		mu.Unlock()
		wg.Done()
	})

	// Publish multiple messages
	pubsub.Publish(ctx, "topic", "msg1")
	pubsub.Publish(ctx, "topic", "msg2")
	pubsub.Publish(ctx, "topic", "msg3")

	wg.Wait()
	assert.Equals(len(messages), 3)
}

func TestFakePubSubProvider_ContextCancellation(t *testing.T) {
	assert := test.NewAssertions(t)

	pubsub := NewFakePubSubProvider()

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Operations should still work (fake provider doesn't check context)
	var wg sync.WaitGroup
	wg.Add(1)

	pubsub.Subscribe(ctx, "topic", func(ctx context.Context, message string) {
		wg.Done()
	})

	err := pubsub.Publish(ctx, "topic", "message")
	assert.Nil(err)

	wg.Wait()
}

func TestFakePubSubProvider_CountersAcrossTopics(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	pubsub := NewFakePubSubProvider().(*FakePubSubProvider)

	// Subscribe to multiple topics
	pubsub.Subscribe(ctx, "topic1", func(ctx context.Context, message string) {})
	pubsub.Subscribe(ctx, "topic2", func(ctx context.Context, message string) {})

	// Publish to different topics
	pubsub.Publish(ctx, "topic1", "msg1")
	pubsub.Publish(ctx, "topic1", "msg2")
	pubsub.Publish(ctx, "topic2", "msg3")

	// Give goroutines time to execute
	time.Sleep(20 * time.Millisecond)

	// Verify counters are independent
	assert.Equals(pubsub.Sent("topic1"), 2)
	assert.Equals(pubsub.Sent("topic2"), 1)
	assert.Equals(pubsub.Received("topic1"), 2)
	assert.Equals(pubsub.Received("topic2"), 1)
}

func TestFakePubSubProvider_InterfaceCompliance(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	pubsub := NewFakePubSubProvider()

	// Verify all interface methods exist and work
	assert.Nil(pubsub.Init())
	assert.Nil(pubsub.Ping())

	// Subscribe
	var wg sync.WaitGroup
	wg.Add(1)
	pubsub.Subscribe(ctx, "test", func(ctx context.Context, message string) {
		wg.Done()
	})

	// Publish
	err := pubsub.Publish(ctx, "test", "message")
	assert.Nil(err)

	wg.Wait()
}

// NOTE: FakePubSubProvider is NOT thread-safe for concurrent Publish/Subscribe
// operations on the same topic. The maps are not protected by mutexes.
// This is acceptable for single-threaded test scenarios.
