package server

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
)

type Server struct {
	Host string `json:"host"`
	Port uint   `json:"port"`

	websocketHub *WebsocketHub
	assets       embed.FS
}

func NewServer(mcb func(message *Message), assets embed.FS) *Server {
	a := &Server{
		Host:         "localhost",
		Port:         4788,
		websocketHub: NewWebsocketHub(),
		assets:       assets,
	}
	a.websocketHub.onReceive = mcb

	return a
}

func (a *Server) Load() {
	a.setRoutes()
}

func (a *Server) setRoutes() {
	var staticFS = fs.FS(a.assets)
	htmlContent, err := fs.Sub(staticFS, "public")
	if err != nil {
		log.Fatal(err)
	}

	// Serve static files
	http.Handle("/", http.FileServer(http.FS(htmlContent)))
	http.HandleFunc("/ws", a.websocketEndpoint)
}

func (a *Server) Start() error {
	fmt.Println("Listening on: http://" + a.Address())
	go a.websocketHub.run()

	return http.ListenAndServe(a.Address(), nil)
}

func (a *Server) Address() string {
	return fmt.Sprintf("%s:%d", a.Host, a.Port)
}

func (a *Server) Hub() *WebsocketHub {
	return a.websocketHub
}

func (a *Server) websocketEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("failed to upgrade websocket connection: %s\n", err.Error())
		return
	}
	client := &WebsocketClient{hub: a.websocketHub, conn: conn, send: make(chan []byte, maxMessageSize)}
	a.websocketHub.register <- client

	go client.writePump()
	go client.readPump()
}
