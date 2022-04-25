package wss

type ClientMessage struct {
	Client  *Client
	Message []byte
}

type Hub struct {
	clients            map[*Client]bool
	register           chan *Client
	unregister         chan *Client
	Broadcast          chan []byte
	Receiver           chan *ClientMessage
	RegisterReceiver   chan *ClientMessage
	UnregisterReceiver chan *ClientMessage
}

func NewHub() *Hub {
	return &Hub{
		clients:            make(map[*Client]bool),
		register:           make(chan *Client),
		unregister:         make(chan *Client),
		Broadcast:          make(chan []byte),
		Receiver:           make(chan *ClientMessage),
		RegisterReceiver:   make(chan *ClientMessage),
		UnregisterReceiver: make(chan *ClientMessage),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			cm := new(ClientMessage)
			cm.Client = client
			h.RegisterReceiver <- cm
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				cm := new(ClientMessage)
				cm.Client = client
				h.UnregisterReceiver <- cm
				delete(h.clients, client)
				close(client.Send)
			}
		case message := <-h.Broadcast:
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
		}
	}
}
