
package pkg

import (
	"fmt"
	"errors"
	"github.com/google/uuid"
)

const (
	HubCmdRemoveSub int    = 0
	HubCmdMessage          = 1
	HubCmdShutdown         = 2
	HubCmdRemoveSubscriber = 3
)

type HubChannel struct {
	Id string
	ClientPings chan Empty
	MsgCh chan string
}

type HubSubscription struct {
	Name string
}

type HubCommand struct {
	CmdType int
	Subscription string
	Message string
	SubscriberId string
}

type Hub struct {
	Ids map[string]bool
	Subscribers map[string][]*HubChannel
	Subscriptions map[string]*HubSubscription
	CommandChSize int
	CommandCh chan HubCommand
	Lock *ReadWriteLock
}

func (hCh *HubChannel) Init() {
	hCh.ClientPings = make(chan Empty)
	hCh.MsgCh = make(chan string)
}

//
// Hub Listen() should check this to see if client has sent
// a ping and is ready for a message.
//
func (hCh *HubChannel) IsClientAlive() bool {
	_, ok := <-hCh.ClientPings
	return ok
}

//
// Clients should close the connection, indicating they're done reading.
//
func (hCh *HubChannel) Close() {
	close(hCh.ClientPings)
}

//
// Clients should send this to indicate they want a message,
// or done indicator.
//
func (hCh *HubChannel) ClientPing() {
	hCh.ClientPings <- Empty{}
}

func (h *Hub) Init() {
	h.Ids = make(map[string]bool)
	h.Subscribers = make(map[string][]*HubChannel)
	h.Subscriptions = make(map[string]*HubSubscription)
	h.CommandCh = make(chan HubCommand, h.CommandChSize)
	h.Lock = &ReadWriteLock{}
	h.Lock.Init()
}

func (h *Hub) CreateSubscription(name string) (*HubSubscription, error) {
	h.Lock.LockForWriting()

	if _, ok := h.Subscriptions[name]; ok {
		h.Lock.WritingUnlock()
		return nil, errors.New(fmt.Sprintf("Subscription already exists with name '%s'", name))
	}

	next := &HubSubscription{Name: name}
	h.Subscriptions[name] = next

	h.Lock.WritingUnlock()
	return next, nil
}

func (h *Hub) GetSubscription(name string) *HubSubscription {
	h.Lock.LockForReading()
	sub := h.Subscriptions[name]
	h.Lock.ReadingUnlock()
	return sub
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

//
// This does not check for subscription existence, as that would
// require a lock and slow things down. If the subscription does not
// exist, the subcribers list will come back empty, and nothing will happen.
//
func (h *Hub) PublishTo(name string, message string) error {
	h.Lock.LockForReading()
	
	if _, ok := h.Subscriptions[name]; !ok {
		h.Lock.ReadingUnlock()
		return errors.New(fmt.Sprintf("Subscription does not exist: %s", name))
	}

	h.Lock.ReadingUnlock()
	// fmt.Printf("PublishTo(): Sending into activity %p\n", h.CommandCh)
	h.CommandCh <- HubCommand{CmdType: HubCmdMessage, Subscription: name, Message: message}
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
			// fmt.Printf("Checking if client is alive from removeSubscription, Id=%s\n", subscriber.Id)
			if subscriber.IsClientAlive() {
				close(subscriber.MsgCh)
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

func (h *Hub) RemoveSubscription(name string) error {
	h.Lock.LockForReading()
	
	if _, ok := h.Subscriptions[name]; !ok {
		h.Lock.ReadingUnlock()
		return errors.New(fmt.Sprintf("Subscription does not exist: %s", name))
	}

	h.Lock.ReadingUnlock()
	h.CommandCh <- HubCommand{CmdType: HubCmdRemoveSub, Subscription: name}

	return nil
}

func (h *Hub) Listen() {
	Loop:
	for {
		// fmt.Printf("Listen(): reading an activity %p\n", h.CommandCh)
		hubCommand := <-h.CommandCh

		switch hubCommand.CmdType {
		case HubCmdShutdown:
			fmt.Printf("Hub got Shutdown\n")
			break Loop
		case HubCmdRemoveSub:
			err := h.removeSubscription(hubCommand.Subscription, false)
			if err != nil {
				fmt.Printf("Error when removing subscription: %s\n", err)
			}
		case HubCmdRemoveSubscriber:
			// fmt.Printf("Removing subscriber from feed message\n")
			err := h.removeSubscriber(hubCommand.Subscription, hubCommand.SubscriberId)
			if err != nil {
				fmt.Printf("Error when removing subscriber: %s\n", err)
			}
		case HubCmdMessage:
			for _, subscriber := range h.SubscribersFor(hubCommand.Subscription) {
				// fmt.Printf("Publishing to client %s\n", subscriber.Id)
				if !subscriber.IsClientAlive() {					
					// fmt.Printf("  Continuing because client is dead\n")
					err := h.removeSubscriber(hubCommand.Subscription, subscriber.Id)
					if err != nil {
						fmt.Printf("Error when removing subscriber: %s\n", err)
					}
					continue
				}
				subscriber.MsgCh <- hubCommand.Message
				// fmt.Printf("Done publishing to client %s\n", subscriber.Id)
			}
		}

		// fmt.Printf("Looping in Listen()\n")
	}

	// fmt.Printf("Removing all subscriptions\n")
	h.removeAllSubscriptions()
}

func (h *Hub) removeSubscriber(name string, id string) error {
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
		// This can happen if a subscriber is removed during publishing and
		// before its HubCmdRemoveSubscriber is treated.
		h.Lock.WritingUnlock()
		return nil
	}

	delete(h.Ids, id)
	h.Subscribers[name] = append(h.Subscribers[name][:idx], h.Subscribers[name][idx+1:]...)
	h.Lock.WritingUnlock()
	return nil
}

