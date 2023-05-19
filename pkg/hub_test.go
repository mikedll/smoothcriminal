
package pkg

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestHubChannel(t *testing.T) {
	hCh := HubChannel{}
	hCh.Init()

	g1 := make(chan bool)
	g2 := make(chan bool)
	
	// Publisher
	go func() {
		if hCh.IsClientAlive() {
			hCh.Send("Hello")
		}
		g1 <- true
	}()

	// Client
	go func() {
		hCh.ClientPing()
		assert.Equal(t, hCh.Done(), false)
		assert.Equal(t, hCh.Read(), "Hello")
		g2 <- true
	}()

	<-g1
	<-g2
}

func TestHubChannelClose(t *testing.T) {
	hCh := HubChannel{}
	hCh.Init()

	g1 := make(chan bool)
	g2 := make(chan bool)
	
	// Publisher
	go func() {
		ok := true
		if hCh.IsClientAlive() {
			hCh.Send("should not be sent")
			ok = false
		}
		g1 <- ok
	}()

	// Client
	go func() {
		hCh.Close()
		g2 <- true
	}()

	assert.Equal(t, <-g1, true)
	<-g2
}

func TestHub(t *testing.T) {
	hub := &Hub{}
	hub.Init()

	hub.CreateSubscription("job:1")

	messageBox := hub.GetSubscription("job:1")
	t.Logf("Some sub, done=%t, value=%s", messageBox.Done(), messageBox.Read())
}
