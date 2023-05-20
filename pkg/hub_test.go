
package pkg

import (
	_ "fmt"
	"errors"
	"testing"
	"time"
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
		assert.Equal(t, false, hCh.Done())
		assert.Equal(t, "Hello", hCh.Read())
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
	assert.Equal(t, "job:1", sub1.Name)
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
		assert.Equal(t, HubActMessage, act.ActType)
		assert.Equal(t, "job:1", act.Subscription)
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

	assert.Equal(t, cli1.Id, subscribers[0].Id)
	assert.Equal(t, cli2.Id, subscribers[1].Id)

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

func TestPublishToNonexistent(t *testing.T) {
	hub := &Hub{}
	hub.Init()
	
	err := hub.PublishTo("job:1", "Hello Mike")
	assert.Equal(t, errors.New("Subscription does not exist: job:1"), err)
}

func TestListenMissedMessage(t *testing.T) {
	hub := &Hub{}
	hub.Init()

	g1 := make(chan bool)
	g2 := make(chan bool)
	g3 := make(chan bool)

	seekSubscription := make(chan bool)
	subscriptionExists := make(chan bool)
	detectMissingSubscription := make(chan bool)
	
	// Listener
	go func() {
		hub.Listen()
		g1 <- true
	}()
	
	// Job
	go func() {
		hub.CreateSubscription("job:1")

		seekSubscription <- true
		<-subscriptionExists
		
		err := hub.PublishTo("job:1", "Hello Mike")
		assert.Nil(t, err)
		err = hub.PublishTo("job:1", "Hello Carol")
		assert.Nil(t, err)		

		err = hub.RemoveSubscription("job:1")
		assert.Nil(t, err)

		detectMissingSubscription <- true
		
		g2 <- true
	}()

	// Client 1
	go func() {
		<-seekSubscription

		for {
			relax, _ := time.ParseDuration("10ms")
			time.Sleep(relax)
			sub := hub.GetSubscription("job:1")
			if sub != nil {
				break
			}
		}

		subscriptionExists <- true
		<-detectMissingSubscription

		for {
			relax, _ := time.ParseDuration("10ms")
			time.Sleep(relax)
			sub := hub.GetSubscription("job:1")
			if sub == nil {
				break
			}
		}		
		
		_, err := hub.Subscribe("job:1")
		assert.Equal(t, errors.New("Subscription does not exist: job:1"), err)

		g3 <- true
	}()

	<-g2
	hub.ActivityFeed <- HubActivity{ActType: HubActShutdown}
	
	<-g1
	<-g3
}

func TestListen(t *testing.T) {
	hub := &Hub{}
	hub.Init()

	g1 := make(chan bool)
	jobDone := make(chan bool)
	g3 := make(chan bool)
	
	// Listener
	go func() {
		hub.Listen()
		g1 <- true
	}()

	clientGo := make(chan bool)
	jobContinue := make(chan bool)
	
	// Job
	go func() {		
		hub.CreateSubscription("job:1")

		clientGo <- true
		<-jobContinue
		
		hub.PublishTo("job:1", "Hello Mike")		
		hub.PublishTo("job:1", "Hello Carol")		
		jobDone <- true
	}()

	// Client 1
	go func() {
		<-clientGo
		cli, err := hub.Subscribe("job:1")
		if err != nil {
			t.Fatalf("Failed to subscribe to job:1")
		}

		jobContinue <- true

		msges := []string{}

		cli.ClientPing()
		if !cli.Done() {
			msges = append(msges, cli.Read())
		}

		cli.ClientPing()
		if !cli.Done() {
			msges = append(msges, cli.Read())
		}		
		
		assert.Equal(t, []string{"Hello Mike", "Hello Carol"}, msges)

		cli.ClientPing()
		assert.True(t, cli.Done())
		
		g3 <- true
	}()
	
	<-jobDone
	hub.ActivityFeed <- HubActivity{ActType: HubActShutdown}
	
	<-g1
	<-g3

	assert.Empty(t, hub.Subscribers)
}

func typicalClient(t *testing.T, hub *Hub, jobContinue chan<- bool, expectedReads []string, exitCh chan<- bool) {	
	cli, err := hub.Subscribe("job:1")
	assert.Nil(t, err)
	
	jobContinue <- true

	messages := []string{}
	for {
		cli.ClientPing()
		if(!cli.Done()) {
			messages = append(messages, cli.Read())			
		} else {
			break
		}
	}

	assert.Equal(t, expectedReads, messages)
	
	exitCh <- true
}

func TestListen2Clients(t *testing.T) {
	hub := &Hub{}
	hub.Init()

	g1 := make(chan bool)
	jobThread := make(chan bool)
	g3 := make(chan bool)
	g4 := make(chan bool)

	clientGo := make(chan bool)
	jobContinue := make(chan bool, 2)
	
	// Listener
	go func() {
		hub.Listen()
		g1 <- true
	}()

	// Job
	go func() {
		hub.CreateSubscription("job:1")

		clientGo <- true
		<-jobContinue
		<-jobContinue
		
		hub.PublishTo("job:1", "Hello Mike")		
		hub.PublishTo("job:1", "Hello Carol")		
		jobThread <- true
	}()

	<-clientGo
	go typicalClient(t, hub, jobContinue, []string{"Hello Mike", "Hello Carol"}, g3)
	go typicalClient(t, hub, jobContinue, []string{"Hello Mike", "Hello Carol"}, g4)

	<-jobThread
	hub.ActivityFeed <- HubActivity{ActType: HubActShutdown}
	
	<-g1
	<-g3
	<-g4
}

func TestClientExitEarly(t *testing.T) {
	hub := &Hub{}
	hub.Init()

	g1 := make(chan bool)
	jobThread := make(chan bool)
	g3 := make(chan bool)
	g4 := make(chan bool)
	g5 := make(chan bool)
	
	clientGo := make(chan bool)
	jobContinue := make(chan bool, 1)
	readyForUnsubscribe := make(chan bool)
	
	// Listener
	go func() {
		hub.Listen()
		g1 <- true
	}()

	// Job
	go func() {
		hub.CreateSubscription("job:1")

		clientGo <- true
		<-jobContinue
		<-jobContinue
		<-jobContinue
		
		hub.PublishTo("job:1", "Hello Mike")

		readyForUnsubscribe <- true

		hub.PublishTo("job:1", "Hello Carol")
		
		jobThread <- true
	}()

	<-clientGo

	// 2 Routine clients
	go typicalClient(t, hub, jobContinue, []string{"Hello Mike", "Hello Carol"}, g3)
	go typicalClient(t, hub, jobContinue, []string{"Hello Mike", "Hello Carol"}, g4)

	// Client who unsubscribes after 1st message
	go func () {	
		cli, err := hub.Subscribe("job:1")
		assert.Nil(t, err)
		
		jobContinue <- true

		result := ""
		cli.ClientPing()
		if(!cli.Done()) {
			result = cli.Read()
		}
		assert.Equal(t, "Hello Mike", result)

		<-readyForUnsubscribe
		
		cli.Close()

		g5 <- true
	}()


	<-jobThread
	hub.ActivityFeed <- HubActivity{ActType: HubActShutdown}
	
	<-g1
	<-g3
	<-g4
	<-g5
}
