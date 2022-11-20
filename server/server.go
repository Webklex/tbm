package server

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"tbm/scraper"
	"time"
)

type Server struct {
	Host string `json:"host"`
	Port uint   `json:"port"`

	websocketHub *WebsocketHub
	assets       embed.FS
	template     *template.Template
	mediaDir     string
	state        map[string]interface{}
	mx           sync.RWMutex
}

type ThreadItem struct {
	Tweet scraper.TweetResult
	User  scraper.ConversationUser
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

func (s *Server) Load(mediaDir string) {
	s.mediaDir = mediaDir
	s.setRoutes()
}

func (s *Server) SetState(state map[string]interface{}) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.state = state
}

func (s *Server) AddState(key string, value interface{}) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.state[key] = value
}

func (s *Server) renderHtml(str string) template.HTML {
	return template.HTML(str)
}

func (s *Server) GetState() map[string]interface{} {
	s.mx.RLock()
	defer s.mx.RUnlock()

	return s.state
}

func (s *Server) setRoutes() {
	var staticFS = fs.FS(s.assets)
	htmlContent, err := fs.Sub(staticFS, "static/public")
	if err != nil {
		log.Fatal(err)
	}
	templates, err := fs.Sub(staticFS, "static/template")

	tmpl := template.New("")
	tmpl.Funcs(template.FuncMap{
		"html":     s.renderHtml,
		"GetState": s.GetState,
	})

	tmpl, err = tmpl.ParseFS(templates, "*.tmpl")
	if err != nil {
		log.Fatal(err)
	} else {
		s.template = tmpl
	}

	// Serve static files
	http.Handle("/", http.FileServer(http.FS(htmlContent)))
	http.HandleFunc("/ws", s.websocketEndpoint)
	http.HandleFunc("/media/", s.mediaEndpoint)
	http.HandleFunc("/video/", s.videoEndpoint)
	http.HandleFunc("/state", s.stateEndpoint)
	http.HandleFunc("/thread/", s.threadEndpoint)
}

func (s *Server) Start() error {
	fmt.Println("Listening on: http://" + s.Address())
	go s.websocketHub.run()

	return http.ListenAndServe(s.Address(), nil)
}

func (s *Server) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

func (s *Server) Hub() *WebsocketHub {
	return s.websocketHub
}

func (s *Server) threadEndpoint(w http.ResponseWriter, r *http.Request) {
	_statusId, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/thread/"))
	if _statusId > 0 {
		filename := path.Join(path.Dir(s.mediaDir), fmt.Sprintf("%d.json", _statusId))
		dat, err := os.ReadFile(filename)
		if err == nil {
			cache := &scraper.CachedTweet{}
			if err := json.Unmarshal(dat, cache); err == nil {
				if tmpl := s.template.Lookup("thread"); tmpl != nil {
					title := bluemonday.StripTagsPolicy().Sanitize(cache.Tweet.FullText)
					if len(title) > 16 {
						title = title[0:13] + "..."
					}

					thread := map[string]*ThreadItem{}

					for tweetId, tweet := range cache.Conversation.GlobalObjects.Tweets {
						user, ok := cache.Conversation.GlobalObjects.Users[tweet.UserIdStr]
						if !ok {
							user = scraper.ConversationUser{}
						}

						for _, hashtag := range tweet.Entities.Hashtags {
							re := regexp.MustCompile(`#` + hashtag.Text + `( |$)`)
							tweet.FullText = re.ReplaceAllString(tweet.FullText, `<a class="text-teal-500" href="https://twitter.com/hashtag/`+hashtag.Text+`" target="_blank" rel="noreferrer">#`+hashtag.Text+`</a> `)
						}
						for _, mention := range tweet.Entities.UserMentions {
							re := regexp.MustCompile(`@` + mention.ScreenName + `( |$)`)
							tweet.FullText = re.ReplaceAllString(tweet.FullText, `<a class="text-teal-600" href="https://twitter.com/`+mention.ScreenName+`" target="_blank" rel="noreferrer">@`+mention.ScreenName+`</a> `)
						}
						for _, _url := range tweet.Entities.Urls {
							tweet.FullText = strings.ReplaceAll(tweet.FullText, _url.Url, `<a class="text-yellow-600" href="`+_url.ExpandedUrl+`" target="_blank" rel="noreferrer">`+_url.ExpandedUrl+`</a>`)
						}
						for _, _url := range tweet.Entities.Media {
							tweet.FullText = strings.ReplaceAll(tweet.FullText, _url.Url, ``)
						}

						thread[tweetId] = &ThreadItem{
							Tweet: tweet,
							User:  user,
						}
					}

					if err := tmpl.Execute(w, map[string]interface{}{
						"State":      s.state,
						"Title":      title,
						"Thread":     thread,
						"Tweet":      cache.Tweet,
						"User":       cache.User,
						"TweetIndex": cache.Index,
					}); err != nil {
						fmt.Println(err.Error())
					}
					return
				} else {
					fmt.Println("template not found")
				}
			} else {
				fmt.Println(err.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}

	http.Error(w, "404 status not found", http.StatusNotFound)
}

func (s *Server) stateEndpoint(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(s.GetState())
	if err != nil {
		http.Error(w, "500 invalid configuration", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(b)
}

func (s *Server) videoEndpoint(w http.ResponseWriter, r *http.Request) {
	_mediaId, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/video/"))
	if _mediaId <= 0 {
		// not found
		http.Error(w, "404 image not found", http.StatusNotFound)
		return
	}
	mediaId := fmt.Sprintf("%d", _mediaId)
	mediaFilename := ""

	_ = filepath.Walk(s.mediaDir, func(mediaFilepath string, info os.FileInfo, err error) error {
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

func (s *Server) mediaEndpoint(w http.ResponseWriter, r *http.Request) {
	_mediaId, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/media/"))
	if _mediaId <= 0 {
		// not found
		http.Error(w, "404 image not found", http.StatusNotFound)
		return
	}
	mediaId := fmt.Sprintf("%d", _mediaId)
	mediaFilename := ""

	_ = filepath.Walk(s.mediaDir, func(mediaFilepath string, info os.FileInfo, err error) error {
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

func (s *Server) websocketEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("failed to upgrade websocket connection: %s\n", err.Error())
		return
	}
	client := &WebsocketClient{hub: s.websocketHub, conn: conn, send: make(chan []byte, maxMessageSize)}
	s.websocketHub.register <- client

	go client.writePump()
	go client.readPump()
}
