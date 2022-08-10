package server

// WebsocketHub maintains the set of active clients and broadcasts messages to the
// clients.
type WebsocketHub struct {
	// Registered clients.
	clients map[*WebsocketClient]bool

	// Inbound messages from the clients.
	broadcast chan []byte
	receive   chan *Message

	// Register requests from the clients.
	register chan *WebsocketClient

	// Unregister requests from clients.
	unregister chan *WebsocketClient

	onReceive func(m *Message)
}

type Message struct {
	Content []byte
	Client  *WebsocketClient
}

//
// NewWebsocketHub
// @Description: Create a new WebsocketHub instance
// @return *WebsocketHub
func NewWebsocketHub() *WebsocketHub {
	return &WebsocketHub{
		broadcast:  make(chan []byte),
		receive:    make(chan *Message),
		register:   make(chan *WebsocketClient),
		unregister: make(chan *WebsocketClient),
		clients:    make(map[*WebsocketClient]bool),
		onReceive: func(m *Message) {

		},
	}
}

//
// run
// @Description: Monitor all channels for incoming changes
// @receiver h *WebsocketHub
func (h *WebsocketHub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.receive:
			h.onReceive(message)
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

//
// Broadcast
// @Description: Broadcast a given message to all connected clients
// @receiver h *WebsocketHub
// @param message []byte
func (h *WebsocketHub) Broadcast(message []byte) {
	h.broadcast <- message
}
