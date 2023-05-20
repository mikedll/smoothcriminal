
package pkg

import (
	"fmt"
	"errors"
	"github.com/google/uuid"
)

const (
	HubActRemoveSub int = 0
	HubActMessage       = 1
	HubActShutdown      = 2
)

type HubChannel struct {
	Id string
	ClientPings chan bool
	DoneCh chan bool
	MsgCh chan string
}

type HubSubscription struct {
	Name string
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
	Lock *ReadWriteLock
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

func (h *Hub) Init() {
	h.Ids = make(map[string]bool)
	h.Subscribers = make(map[string][]*HubChannel)
	h.Subscriptions = make(map[string]*HubSubscription)
	h.ActivityFeed = make(chan HubActivity)
	h.Lock = &ReadWriteLock{}
	h.Lock.Init()
}

// TODO: Guard with semaphore
// TODO: Return error if subscription exists
func (h *Hub) CreateSubscription(name string) *HubSubscription {
	next := &HubSubscription{Name: name}
	h.Subscriptions[name] = next
	return next
}

// TODO: Guard with semaphore
func (h *Hub) GetSubscription(name string) *HubSubscription {
	return h.Subscriptions[name]
}

func (h *Hub) Subscribe(name string) (*HubChannel, error) {
	h.Lock.LockForWriting()
	
	if _, ok := h.Subscriptions[name]; !ok {
		h.Lock.WritingUnlock()
		return nil, errors.New(fmt.Sprintf("Subscription does not exist: %s", name))
	}

	if _, ok := h.Subscribers[name]; !ok {
		h.Subscribers[name] = []*HubChannel{}
	}

	nextUUID := uuid.New()
	if _, ok := h.Ids[nextUUID.String()]; ok {
		h.Lock.WritingUnlock()
		return nil, errors.New("UUID collision")
	}

	h.Ids[nextUUID.String()] = true
	next := &HubChannel{Id: nextUUID.String()}
	next.Init()
	h.Subscribers[name] = append(h.Subscribers[name], next)

	h.Lock.WritingUnlock()
	return next, nil
}

func (h *Hub) SubscribersFor(name string) []*HubChannel {
	h.Lock.LockForReading()
	if _, ok := h.Subscribers[name]; !ok {
		h.Lock.ReadingUnlock()
		return []*HubChannel{}
	}

	cpy := append([]*HubChannel{}, h.Subscribers[name]...)
	h.Lock.ReadingUnlock()
	return cpy
}

// TODO: Protect with lock
//
// This does not check for subscription existence, as that would
// require a lock and slow things down. If the subscription does not
// exist, the subcribers list will come back empty, and nothing will happen.
//
func (h *Hub) PublishTo(name string, message string) error {
	if _, ok := h.Subscriptions[name]; !ok {
		return errors.New(fmt.Sprintf("Subscription does not exist: %s", name))
	}
	// fmt.Printf("PublishTo(): Sending into activity %p\n", h.ActivityFeed)
	h.ActivityFeed <- HubActivity{ActType: HubActMessage, Subscription: name, Message: message}
	return nil
}

func (h *Hub) removeSubscription(name string, alreadyLocked bool) error {
	if !alreadyLocked {
		h.Lock.LockForWriting()
	}
	
	if _, ok := h.Subscriptions[name]; !ok {
		if !alreadyLocked {
			h.Lock.WritingUnlock()
		}
		return errors.New(fmt.Sprintf("Subscription does not exist: %s", name))
	}
	
	if _, ok := h.Subscribers[name]; ok {
		for _, subscriber := range h.Subscribers[name] {
			if subscriber.IsClientAlive() {
				subscriber.DoneCh <- true
			}
		}

		delete(h.Subscribers, name)
	}

	delete(h.Subscriptions, name)
	if !alreadyLocked {
		h.Lock.WritingUnlock()
	}
	return nil
}

func (h *Hub) removeAllSubscriptions() {
	h.Lock.LockForWriting()

	for name, _ := range h.Subscriptions {
		err := h.removeSubscription(name, true)
		if err != nil {
			fmt.Printf("Error when removing subscription %s: %s", name, err)
		}
	}

	h.Lock.WritingUnlock()
}

// TODO: Protect with locks
func (h *Hub) RemoveSubscription(name string) error {
	if _, ok := h.Subscriptions[name]; !ok {
		return errors.New(fmt.Sprintf("Subscription does not exist: %s", name))
	}

	h.ActivityFeed <- HubActivity{ActType: HubActRemoveSub, Subscription: name}

	return nil
}

func (h *Hub) Listen() {
	Loop:
	for {
		// this has to be generalized to an action feed, and the messages have to
		// state what they're describing. this way, the subscribers can be cleaned
		// up right away, instead of having to wait to see if there was a message.
		// subscribers to topics should be followed with integer ids, so that we
		// can see if we already removed them off.

		// TODO: Get rid of Subscription struct. It doesn't get us anything. We
		// can just use the activity stream.
		// fmt.Printf("Listen(): reading an activity %p\n", h.ActivityFeed)
		hubActivity := <-h.ActivityFeed

		switch hubActivity.ActType {
		case HubActShutdown:
			fmt.Printf("Hub got Shutdown\n")
			break Loop
		case HubActRemoveSub:
			err := h.removeSubscription(hubActivity.Subscription, false)
			if err != nil {
				fmt.Printf("Error when removing subscription: %s\n", err)
			}
		case HubActMessage:
			for _, subscriber := range h.SubscribersFor(hubActivity.Subscription) {
				// fmt.Printf("Publishing to client %s\n", subscriber.Id)
				if !subscriber.IsClientAlive() {
					// this has to be the only way for a Client to disconnect,
					// which means some clients will stick around until a subscription
					// is over, which we just have to live with. they'll be blocking
					// anyway so it's not a big deal. it's nice that there's only
					// one way for a client to abort its connection.
					//
					// we can't have a "Client Disconnected" Activity because it will lead
					// to deadlock. Suppose a Client sends "Client Disconnected" to the
					// Activity feed while this for loop is executing. That send in
					// the Client will block because Listen() is busy here,
					// and IsClientAlive will also block because the client is
					// waiting for Listen() to pickup its message.
					//
					// Rebuttal to Preceding, Todo: Use a buffered channel. Permit Clients
					// to move on with their usual loop if channel is full, but take advantage
					// of the optimization if we can.
					
					// fmt.Printf("  Continuing because client is dead\n")
					err := h.RemoveSubscriber(hubActivity.Subscription, subscriber.Id)
					if err != nil {
						fmt.Printf("Error when removing subscriber: %s", err)
					}
					continue
				}
				subscriber.DoneCh <- false
				subscriber.MsgCh <- hubActivity.Message
				// fmt.Printf("Done publishing to client %s\n", subscriber.Id)
			}
		}
	}

	h.removeAllSubscriptions()
}

func (h *Hub) RemoveSubscriber(name string, id string) error {
	h.Lock.LockForWriting()
	
	if _, ok := h.Subscriptions[name]; !ok {
		h.Lock.WritingUnlock()
		return errors.New(fmt.Sprintf("Subscription does not exist: %s", name))
	}

	idx := -1
	for i, subscriber := range h.Subscribers[name] {
		if subscriber.Id == id {
			idx = i
		}
	}

	if idx == -1 {
		h.Lock.WritingUnlock()
		return errors.New(fmt.Sprintf("Unable to find subscriber with id: %s", id))
	}

	delete(h.Ids, id)
	h.Subscribers[name] = append(h.Subscribers[name][:idx], h.Subscribers[name][idx+1:]...)
	h.Lock.WritingUnlock()
	return nil
}

