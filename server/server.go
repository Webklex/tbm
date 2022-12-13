package server

import (
	"embed"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"tbm/server/response"
	"tbm/utils/log"
	"time"
)

type Server struct {
	Host string `json:"host"`
	Port uint   `json:"port"`

	websocketHub *WebsocketHub
	assets       embed.FS
	MediaDir     string `json:"-"`
	router       *httprouter.Router
	template     *template.Template
	mx           sync.RWMutex
	funcMap      template.FuncMap
	server       *http.Server
}

type HttpCallback func(w http.ResponseWriter, r *http.Request, p httprouter.Params) *response.Error

func NewServer(mcb func(message *Message), assets embed.FS, funcMap template.FuncMap) *Server {
	s := &Server{
		Host:         "localhost",
		Port:         4788,
		websocketHub: NewWebsocketHub(),
		assets:       assets,
		funcMap:      funcMap,
		router:       httprouter.New(),
	}
	s.websocketHub.onReceive = mcb
	s.funcMap["html"] = s.renderHtml

	return s
}

func (s *Server) Route(route func(r *httprouter.Router)) {
	route(s.router)
}

func (s *Server) Load() {
	s.setRoutes()
}

func (s *Server) Template() *template.Template {
	return s.template
}

func (s *Server) renderHtml(str string) template.HTML {
	return template.HTML(str)
}

func (s *Server) setRoutes() {
	var staticFS = fs.FS(s.assets)
	htmlContent, err := fs.Sub(staticFS, "static/public")
	if err != nil {
		log.Fatal(err)
	}
	templates, err := fs.Sub(staticFS, "static/template")

	tmpl := template.New("")
	tmpl.Funcs(s.funcMap)

	tmpl, err = tmpl.ParseFS(templates, "*.tmpl")
	if err != nil {
		log.Fatal(err)
	} else {
		s.template = tmpl
	}

	s.router.NotFound = http.FileServer(http.FS(htmlContent))

	s.router.GET("/ws", s.websocketEndpoint)
	s.router.GET("/media/:id", s.CreateHandler(s.mediaEndpoint))
	s.router.GET("/video/:id", s.CreateHandler(s.videoEndpoint))
}

func (s *Server) Start() error {
	log.Info("Server started on: http://%s", s.Address())
	go s.websocketHub.run()

	s.server = &http.Server{Addr: s.Address(), Handler: s.router}
	go s.server.ListenAndServe()

	return nil
}

func (s *Server) Stop() error {
	s.websocketHub.Close()
	if s.server != nil {
		if err := s.server.Close(); err != nil {
			return err
		}
		s.server = nil
	}
	log.Warning("Server stopped")

	return nil
}

func (s *Server) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

func (s *Server) Hub() *WebsocketHub {
	return s.websocketHub
}

func (s *Server) websocketEndpoint(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("Failed to upgrade websocket connection: %s", err.Error())
		return
	}
	client := &WebsocketClient{hub: s.websocketHub, conn: conn, send: make(chan []byte, maxMessageSize)}
	s.websocketHub.register <- client

	go client.writePump()
	go client.readPump()
}

func (s *Server) CreateHandler(handler HttpCallback) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if err := handler(w, r, p); err != nil {
			http.Error(w, err.Error.Error(), err.Status)
			return
		}
	}
}

func (s *Server) CreateJsonHandler(handler func(resp *response.JsonResponse)) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		resp := response.NewJsonResponse(w, r, p)
		handler(resp.(*response.JsonResponse))
		resp.Render()
	}
}

func (s *Server) CreateViewHandler(name string, handler func(resp *response.ViewResponse)) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if tmpl := s.Template().Lookup(name); tmpl != nil {
			resp := response.NewViewResponse(tmpl, w, r, p)
			handler(resp.(*response.ViewResponse))
			resp.Render()
		} else {
			log.Error("Template not found: %s", name)
		}
	}
}

func (s *Server) videoEndpoint(w http.ResponseWriter, r *http.Request, ps httprouter.Params) *response.Error {
	return s.serveMediaFile([]string{"mp4", "avi", "wav", "gif"}, w, r, ps)
}

func (s *Server) mediaEndpoint(w http.ResponseWriter, r *http.Request, ps httprouter.Params) *response.Error {
	return s.serveMediaFile([]string{"jpg", "jpeg", "png", "gif"}, w, r, ps)
}

//
// serveMediaFile
// @Description: Serve a media file of a given range of allowed extensions
// @receiver a *Application
// @param allowedExtensions []string
// @param w http.ResponseWriter
// @param r *http.Request
// @param ps httprouter.Params
// @return *server.Error
func (s *Server) serveMediaFile(allowedExtensions []string, w http.ResponseWriter, r *http.Request, ps httprouter.Params) *response.Error {
	_mediaId, _ := strconv.Atoi(ps.ByName("id"))
	if _mediaId <= 0 {
		// not found
		return response.NewErrorFromStatus(http.StatusNotFound)
	}
	mediaId := fmt.Sprintf("%d", _mediaId)
	mediaFilename := ""

	_ = filepath.Walk(s.MediaDir, func(mediaFilepath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			ext := filepath.Ext(mediaFilepath)
			if strings.TrimSuffix(filepath.Base(mediaFilepath), ext) == mediaId {
				for _, allowed := range allowedExtensions {
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
			return response.NewErrorFromStatus(http.StatusNotFound)
		}
		defer f.Close()

		http.ServeContent(w, r, mediaFilename, time.Now(), f)
	} else {
		return response.NewErrorFromStatus(http.StatusNotFound)
	}

	return nil
}
