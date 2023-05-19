
package pkg

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestHubChannel(t *testing.T) {
	hCh := HubChannel{}
	hCh.Init()

	// Simulate job manager running and sending to client
	go func() {
		if hCh.IsClientAlive() {
			hCh.Send("Hello")
		}
	}()

	hCh.ClientPing()
	assert.Equal(t, hCh.Done(), false)
	assert.Equal(t, hCh.Read(), "Hello")
}

func TestHubChannelClose(t *testing.T) {
	hCh := HubChannel{}
	hCh.Init()
	
	go func() {
		if hCh.IsClientAlive() {
			hCh.Send("should not appear (and should not block)")
		}
	}()

	hCh.Close()

	// expect to finish
}

func TestHub(t *testing.T) {
	hub := &Hub{}
	hub.Init()

	hub.CreateSubscription("job:1")

	messageBox := hub.GetSubscription("job:1")
	t.Logf("Some sub, done=%t, value=%s", messageBox.Done(), messageBox.Read())
}
