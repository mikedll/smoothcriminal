
package pkg

type HubChannel struct {
	name string
	clientPings chan bool
	msgCh chan string
	doneCh chan bool
}

type Hub struct {
	subscriptions map[string]*HubChannel
}

func (hCh *HubChannel) Init() {
	hCh.clientPings = make(chan bool)
	hCh.doneCh = make(chan bool)
	hCh.msgCh = make(chan string)
}

//
// Job manager should check this to see if client has sent
// a ping and is ready for a message.
//
func (hCh *HubChannel) IsClientAlive() bool {
	return <-hCh.clientPings
}

//
// Clients should close the connection, indicating they're done reading.
//
func (hCh *HubChannel) Close() {
	hCh.clientPings <- false
}

//
// Clients should send this to indicate they want a message,
// or done indicator.
//
func (hCh *HubChannel) ClientPing() {
	hCh.clientPings <- true
}

func (hCh *HubChannel) Send(message string) {
	hCh.doneCh <- false
	hCh.msgCh <- message
}

//
// For clients to read. Indicates there are no more messages coming.
//
func (hCh *HubChannel) Done() bool {
	return <-hCh.doneCh
}

func(hCh *HubChannel) Read() string {
	return <-hCh.msgCh
}

func (h *Hub) Init() {
	h.subscriptions = make(map[string]*HubChannel)
}

func (h *Hub) CreateSubscription(name string) {
	next := &HubChannel{name: name}
	next.Init()
	h.subscriptions[name] = next
}

func (h *Hub) GetSubscription(name string) *HubChannel {
	return h.subscriptions[name]
}
