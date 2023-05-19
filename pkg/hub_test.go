
package pkg

import (
	"errors"
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

func TestGetSubscription(t *testing.T) {
	hub := &Hub{}
	hub.Init()

	sub1 := hub.CreateSubscription("job:1")

	hubSub := hub.GetSubscription("job:1")

	assert.IsType(t, &HubSubscription{}, hubSub)
	assert.Equal(t, sub1.Name, "job:1")
	assert.Equal(t, sub1, hubSub)
}

func TestActivityFeed(t *testing.T) {
	hub := &Hub{}
	hub.Init()

	g1 := make(chan bool)
	g2 := make(chan bool)
	
	// Listener
	go func() {
		act := <- hub.ActivityFeed
		assert.Equal(t, act.ActType, HubActMessage)
		assert.Equal(t, act.Subscription, "job:1")
		g1 <- true
	}()

	// Job
	go func() {
		hub.ActivityFeed <- HubActivity{ActType: HubActMessage, Subscription: "job:1", Message: "Hello Mike"}
		g2 <- true
	}()

	<-g1
	<-g2
}

func TestGetSubscribers(t *testing.T) {
	hub := &Hub{}
	hub.Init()

	initialSubscribers := hub.SubscribersFor("job:1")
	assert.Empty(t, initialSubscribers)
	
	hub.CreateSubscription("job:1")

	cli1, err := hub.Subscribe("job:1")
	assert.Nil(t, err)
	
	cli2, err := hub.Subscribe("job:1")
	assert.Nil(t, err)

	subscribers := hub.SubscribersFor("job:1")

	assert.Equal(t, subscribers[0].Id, cli1.Id)
	assert.Equal(t, subscribers[1].Id, cli2.Id)

	if _, ok := hub.Ids[cli1.Id]; !ok {
		t.Fatalf("Failed to track key: %s\n", cli1.Id)
	}

	if _, ok := hub.Ids[cli2.Id]; !ok {
		t.Fatalf("Failed to track key: %s\n", cli2.Id)
	}
}

func TestRemoveSubscriber(t *testing.T) {
	hub := &Hub{}
	hub.Init()

	hub.CreateSubscription("job:1")

	cli1, err := hub.Subscribe("job:1")
	assert.Nil(t, err)
	
	cli2, err := hub.Subscribe("job:1")
	assert.Nil(t, err)

	cli3, err := hub.Subscribe("job:1")
	assert.Nil(t, err)

	err = hub.RemoveSubscriber("job:1", "blah")
	assert.Equal(t, err, errors.New("Unable to find subscriber with id: blah"))
	
	err = hub.RemoveSubscriber("job:1", cli2.Id)
	assert.Nil(t, err)
	subscribersAfter := hub.SubscribersFor("job:1")
	assert.Equal(t, []*HubChannel{cli1, cli3}, subscribersAfter)

	err = hub.RemoveSubscriber("job:1", cli1.Id)
	assert.Nil(t, err)
	subscribersAfter = hub.SubscribersFor("job:1")
	assert.Equal(t, []*HubChannel{cli3}, subscribersAfter)

	err = hub.RemoveSubscriber("job:1", cli3.Id)
	assert.Nil(t, err)
	subscribersAfter = hub.SubscribersFor("job:1")
	assert.Empty(t, subscribersAfter)	
}

func TestListen(t *testing.T) {
	hub := &Hub{}
	hub.Init()

	g1 := make(chan bool)
	g2 := make(chan bool)

	// Listener
	go func() {
		hub.Listen()
		g1 <- true
	}()

	// Job
	go func() {
		hub.PublishTo("job:1", "no subject")
		hub.CreateSubscription("job:1")
		hub.PublishTo("job:1", "something")
		hub.ActivityFeed <- HubActivity{ActType: HubActShutdown}
		g2 <- true
	}()

	<-g1
	<-g2
}
