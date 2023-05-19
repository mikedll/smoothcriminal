
package pkg

type HubChannel struct {
	name string
	msgCh chan string
	doneCh chan bool
}

type Hub struct {
	subscriptions map[string]*HubChannel
}

func (hCh *HubChannel) Init() {
	hCh.msgCh = make(chan string)
	hCh.doneCh = make(chan bool)
}

func (hCh *HubChannel) Send(message string) {
	hCh.doneCh <- false
	hCh.msgCh <- message
}

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
