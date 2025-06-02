package pubsub_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"my-project/infrastructure/pubsub"
)

// TestNewTestPubSub tests the creation of a new TestPubSub
func TestNewTestPubSub(t *testing.T) {
	// This is a simple test to ensure the function exists and returns an object
	// We can't do much more without mocking the Google Cloud PubSub client
	testPubSub := pubsub.NewTestPubSub(nil)
	assert.NotNil(t, testPubSub)
}