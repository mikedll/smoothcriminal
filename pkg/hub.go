
package pkg

import (
	"fmt"
	"errors"
	"github.com/google/uuid"
)

const (
	HubActClose int = 0
	HubActMessage   = 1
)

type HubChannel struct {
	Id string
	Name string
	ClientPings chan bool
	DoneCh chan bool
	MsgCh chan string
}

type HubSubscription struct {
	Name string
	MsgCh chan string
}

type HubActivity struct {
	ActType int
	Subscription string
	Message string
}

type Hub struct {
	Ids map[string]bool
	Subscribers map[string][]*HubChannel
	Subscriptions map[string]*HubSubscription
	ActivityFeed chan HubActivity
}

func (hCh *HubChannel) Init() {
	hCh.ClientPings = make(chan bool)
	hCh.DoneCh = make(chan bool)
	hCh.MsgCh = make(chan string)
}

//
// Job manager should check this to see if client has sent
// a ping and is ready for a message.
//
func (hCh *HubChannel) IsClientAlive() bool {
	return <-hCh.ClientPings
}

//
// Clients should close the connection, indicating they're done reading.
//
func (hCh *HubChannel) Close() {
	hCh.ClientPings <- false
}

//
// Clients should send this to indicate they want a message,
// or done indicator.
//
func (hCh *HubChannel) ClientPing() {
	hCh.ClientPings <- true
}

func (hCh *HubChannel) Send(message string) {
	hCh.DoneCh <- false
	hCh.MsgCh <- message
}

//
// For clients to read. Indicates there are no more messages coming.
//
func (hCh *HubChannel) Done() bool {
	return <-hCh.DoneCh
}

func(hCh *HubChannel) Read() string {
	return <-hCh.MsgCh
}

func (hSub *HubSubscription) Init() {
	hSub.MsgCh = make(chan string)
}

func (h *Hub) Init() {
	h.Ids = make(map[string]bool)
	h.Subscribers = make(map[string][]*HubChannel)
	h.Subscriptions = make(map[string]*HubSubscription)
	h.ActivityFeed = make(chan HubActivity)
}

// TODO: Guard with semaphore
func (h *Hub) CreateSubscription(name string) *HubSubscription {
	next := &HubSubscription{Name: name}
	next.Init()
	h.Subscriptions[name] = next
	return next
}

// TODO: Guard with semaphore
func (h *Hub) GetSubscription(name string) *HubSubscription {
	return h.Subscriptions[name]
}

// TODO: Guard with semaphore
func (h *Hub) Subscribe(name string) (*HubChannel, error) {
	if _, ok := h.Subscriptions[name]; !ok {
		return nil, errors.New("no such subscription")
	}

	if _, ok := h.Subscribers[name]; !ok {
		h.Subscribers[name] = []*HubChannel{}
	}

	nextUUID := uuid.New()
	if _, ok := h.Ids[nextUUID.String()]; ok {
		return nil, errors.New("UUID collision")
	}

	h.Ids[nextUUID.String()] = true
	next := &HubChannel{Id: nextUUID.String()}
	h.Subscribers[name] = append(h.Subscribers[name], next)

	return next, nil
}

// TODO: Guard with semaphore
func (h *Hub) SubscribersFor(name string) []*HubChannel {
	if _, ok := h.Subscribers[name]; !ok {
		return []*HubChannel{}
	}

	return h.Subscribers[name]
}

func (h *Hub) PublishTo(name string, message string) {
	// Should send to activity feed as a message activity
	// Should send to subscription message channel
}

func (h *Hub) Listen() {
	for {
		// this has to be generalized to an action feed, and the messages have to
		// state what they're describing. this way, the subscribers can be cleaned
		// up right away, instead of having to wait to see if there was a message.
		// subscribers to topics should be followed with integer ids, so that we
		// can see if we already removed them off.

		// TODO: Get rid of Subscription struct. It doesn't get us anything. We
		// can just use the activity stream.
		hubActivity := <-h.ActivityFeed
		
		// TODO: Handle subscription is done

		if hubActivity.ActType == HubActMessage {
			for _, subscriber := range h.SubscribersFor(hubActivity.Subscription) {
				if !subscriber.IsClientAlive() {
					// this can potentially beat the activityFeed message, but the activity
					// feed should just fail to find the subscriber in such a case, and continue.
					// we should delete this person from the subscribers list.
					continue
				}
				subscriber.DoneCh <- false
				subscriber.MsgCh <- hubActivity.Message
			}
		}
	}
}

// TODO: Guard with semaphore
func (h *Hub) RemoveSubscriber(name string, id string) error {
	if _, ok := h.Subscriptions[name]; !ok {
		return errors.New(fmt.Sprintf("No such subscription: %s", name))
	}

	idx := -1
	for i, subscriber := range h.SubscribersFor(name) {
		if subscriber.Id == id {
			idx = i
		}
	}

	if idx == -1 {
		return errors.New(fmt.Sprintf("Unable to find subscriber with id: %s", id))
	}

	h.Subscribers[name] = append(h.Subscribers[name][:idx], h.Subscribers[name][idx+1:]...)
	return nil
}
