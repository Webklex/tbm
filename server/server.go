package server

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Server struct {
	Host string `json:"host"`
	Port uint   `json:"port"`

	websocketHub *WebsocketHub
	assets       embed.FS
	mediaDir     string
	state        map[string]interface{}
	mx           sync.RWMutex
}

func NewServer(mcb func(message *Message), assets embed.FS) *Server {
	a := &Server{
		Host:         "localhost",
		Port:         4788,
		websocketHub: NewWebsocketHub(),
		assets:       assets,
		state:        map[string]interface{}{},
	}
	a.websocketHub.onReceive = mcb

	return a
}

func (a *Server) Load(mediaDir string) {
	a.mediaDir = mediaDir
	a.setRoutes()
}

func (a *Server) SetState(state map[string]interface{}) {
	a.mx.Lock()
	defer a.mx.Unlock()

	a.state = state
}

func (a *Server) AddState(key string, value interface{}) {
	a.mx.Lock()
	defer a.mx.Unlock()

	a.state[key] = value
}

func (a *Server) GetState() map[string]interface{} {
	a.mx.RLock()
	defer a.mx.RUnlock()

	return a.state
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
	http.HandleFunc("/media/", a.mediaEndpoint)
	http.HandleFunc("/video/", a.videoEndpoint)
	http.HandleFunc("/state", a.stateEndpoint)
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

func (a *Server) stateEndpoint(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(a.GetState())
	if err != nil {
		http.Error(w, "500 invalid configuration", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(b)
}

func (a *Server) videoEndpoint(w http.ResponseWriter, r *http.Request) {
	_mediaId, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/video/"))
	if _mediaId <= 0 {
		// not found
		http.Error(w, "404 image not found", http.StatusNotFound)
		return
	}
	mediaId := fmt.Sprintf("%d", _mediaId)
	mediaFilename := ""

	_ = filepath.Walk(a.mediaDir, func(mediaFilepath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			ext := filepath.Ext(mediaFilepath)
			if strings.TrimSuffix(filepath.Base(mediaFilepath), ext) == mediaId {
				for _, allowed := range []string{"mp4", "avi", "wav", "gif"} {
					if allowed == ext[1:] {
						mediaFilename = mediaFilepath
						return errors.New("done")
					}
				}
			}
		}

		return nil
	})

	if mediaFilename != "" {
		f, err := os.Open(mediaFilename)
		if err != nil {
			http.Error(w, "404 image not found", http.StatusNotFound)
			return
		}
		defer f.Close()

		http.ServeContent(w, r, mediaFilename, time.Now(), f)
	} else {
		http.Error(w, "404 image not found", http.StatusNotFound)
	}
}

func (a *Server) mediaEndpoint(w http.ResponseWriter, r *http.Request) {
	_mediaId, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/media/"))
	if _mediaId <= 0 {
		// not found
		http.Error(w, "404 image not found", http.StatusNotFound)
		return
	}
	mediaId := fmt.Sprintf("%d", _mediaId)
	mediaFilename := ""

	_ = filepath.Walk(a.mediaDir, func(mediaFilepath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			ext := filepath.Ext(mediaFilepath)
			if strings.TrimSuffix(filepath.Base(mediaFilepath), ext) == mediaId {
				for _, allowed := range []string{"jpg", "jpeg", "png", "gif"} {
					if allowed == ext[1:] {
						mediaFilename = mediaFilepath
						return errors.New("done")
					}
				}
			}
		}

		return nil
	})

	if mediaFilename != "" {
		f, err := os.Open(mediaFilename)
		if err != nil {
			http.Error(w, "404 image not found", http.StatusNotFound)
			return
		}
		defer f.Close()

		http.ServeContent(w, r, mediaFilename, time.Now(), f)
	} else {
		http.Error(w, "404 image not found", http.StatusNotFound)
	}
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
