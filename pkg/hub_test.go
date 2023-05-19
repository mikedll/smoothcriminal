
package pkg

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestHubChannel(t *testing.T) {
	hCh := HubChannel{}
	hCh.Init()

	hCh.Send("Hello")
	assert.Equal(t, hCh.Done(), false)
}

func TestHub(t *testing.T) {
	hub := &Hub{}
	hub.Init()

	hub.CreateSubscription("job:1")

	messageBox := hub.GetSubscription("job:1")
	t.Logf("Some sub, done=%t, value=%s", messageBox.Done(), messageBox.Read())
}
