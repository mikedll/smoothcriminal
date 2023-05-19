
package pkg

type HubChannel struct {
	Name string
	ClientPings chan bool
	DoneCh chan bool
	MsgCh chan string
}

type Hub struct {
	Subscriptions map[string]*HubChannel
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
	h.Subscriptions = make(map[string]*HubChannel)
}

func (h *Hub) CreateSubscription(name string) {
	next := &HubChannel{Name: name}
	next.Init()
	h.Subscriptions[name] = next
}

func (h *Hub) GetSubscription(name string) *HubChannel {
	return h.Subscriptions[name]
}
