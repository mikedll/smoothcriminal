
package pkg

import (
	"errors"
	"github.com/google/uuid"
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

type Hub struct {
	Ids map[string]bool
	Subscribers map[string][]*HubChannel
	Subscriptions map[string]*HubSubscription
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
}

func (h *Hub) CreateSubscription(name string) *HubSubscription {
	next := &HubSubscription{Name: name}
	next.Init()
	h.Subscriptions[name] = next
	return next
}

func (h *Hub) GetSubscription(name string) *HubSubscription {
	return h.Subscriptions[name]
}

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
