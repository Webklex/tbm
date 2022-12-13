package app

import (
	"embed"
	"encoding/json"
	"errors"
	"github.com/julienschmidt/httprouter"
	"html/template"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"tbm/scraper"
	"tbm/server"
	"tbm/utils/filesystem"
	"tbm/utils/log"
	"time"
)

type Application struct {
	DataDir string          `json:"data_dir"`
	Mode    ApplicationMode `json:"mode"`
	Danger  DangerOptions   `json:"danger"`
	SortBy  string          `json:"sort_by"`

	Build          Build  `json:"-"`
	ConfigFileName string `json:"-"`

	Server  *server.Server   `json:"server"`
	Scraper *scraper.Scraper `json:"scraper"`

	mx     sync.RWMutex
	tweets map[string]*scraper.CachedTweet
	state  map[string]interface{}
}

type Build struct {
	Number  string `json:"number"`
	Version string `json:"version"`
}

type DangerOptions struct {
	RemoveBookmarks bool `json:"remove_bookmarks"`
}

type ApplicationMode string

const (
	OfflineMode ApplicationMode = "offline"
	OnlineMode  ApplicationMode = "online"
)

func (m ApplicationMode) ToString() string {
	return string(m)
}

func NewApplication(assets embed.FS) *Application {
	dir, _ := os.Getwd()

	a := &Application{
		SortBy:         "date",
		DataDir:        path.Join(dir, "data"),
		ConfigFileName: path.Join(dir, "config.json"),
		tweets:         map[string]*scraper.CachedTweet{},
		Mode:           OnlineMode,
		Danger: DangerOptions{
			RemoveBookmarks: false,
		},
		state: map[string]interface{}{},
	}

	a.Scraper = scraper.NewScraper(a.onNewTweet)
	a.Server = server.NewServer(a.websocketCallback, assets, map[string]interface{}{
		"html":       a.renderHtml,
		"GetState":   a.GetState,
		"FormatTime": a.FormatTime,
	})
	a.Server.Route(func(r *httprouter.Router) {
		r.GET("/", a.Server.CreateViewHandler("tweet.index", a.tweetsView))
		r.GET("/status", a.Server.CreateViewHandler("status.index", a.statusView))

		r.GET("/config", a.Server.CreateViewHandler("config.show", a.configView))
		r.POST("/config", a.Server.CreateViewHandler("config.show", a.updateConfigView))

		r.GET("/tweet/:id", a.Server.CreateViewHandler("tweet.show", a.tweetView))

		r.GET("/api/state", a.Server.CreateJsonHandler(a.stateEndpoint))
		r.GET("/api/status", a.Server.CreateJsonHandler(a.statusEndpoint))
		r.GET("/api/tweet", a.Server.CreateJsonHandler(a.tweetsEndpoint))
		r.GET("/api/tweet/:id", a.Server.CreateJsonHandler(a.tweetEndpoint))
	})

	return a
}

func (a *Application) loadConfigFile() error {
	if _, err := os.Stat(a.ConfigFileName); err == nil {
		content, err := ioutil.ReadFile(a.ConfigFileName)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(content, &a); err != nil {
			return err
		}
		if a.Scraper.RawTimeout != "" {
			a.Scraper.Timeout, err = time.ParseDuration(a.Scraper.RawTimeout)
		}
		if a.Scraper.RawDelay != "" {
			a.Scraper.Delay, err = time.ParseDuration(a.Scraper.RawDelay)
		}
	}
	return nil
}

func (a *Application) Load() error {
	if err := a.loadConfigFile(); err != nil {
		return err
	}
	filesystem.CreateDirectory(a.DataDir)
	filesystem.CreateDirectory(path.Join(a.DataDir, "media"))
	a.Server.MediaDir = path.Join(a.DataDir, "media")
	a.Server.Load()
	a.LoadTweetCache()

	return nil
}

func (a *Application) LoadTweetCache() {
	items, _ := ioutil.ReadDir(a.DataDir)
	for _, item := range items {
		if item.IsDir() == false {
			dat, err := os.ReadFile(path.Join(a.DataDir, item.Name()))
			if err == nil {
				ct := &scraper.CachedTweet{}
				if err := json.Unmarshal(dat, ct); err == nil {
					ct.Version = a.Build.Version
					a.tweets[ct.Tweet.IdStr] = ct
				}
			}
		}
	}
}

func (a *Application) Start() error {
	a.AddState("mode", a.Mode)

	if a.Mode == OnlineMode {
		a.Scraper.Start(a.Danger.RemoveBookmarks)
	}

	return a.Server.Start()
}

func (a *Application) Stop() error {
	a.Scraper.Stop()
	return a.Server.Stop()
}

func (a *Application) websocketCallback(m *server.Message) {
	t := &Task{}
	r := NewResponse()
	if err := json.Unmarshal(m.Content, t); err != nil {
		r.SetErrorStr("failed to decode message")
	}

	switch t.Command {
	default:
		r.SetErrorStr("unknown command")
	}

	if b, err := r.Encode(); err == nil {
		m.Client.Send(b)
	} else {
		log.Error("failed to encode response: %s", err.Error())
	}
}

func (a *Application) SearchTweets(query string) []*scraper.CachedTweet {
	query = strings.ToLower(query)
	tweets := make([]*scraper.CachedTweet, 0)
	for _, tweet := range a.GetTweets() {
		add := strings.Contains(strings.ToLower(tweet.Tweet.FullText), query)
		if add == false {
			for _, u := range tweet.Tweet.Entities.Urls {
				if strings.Contains(strings.ToLower(u.ExpandedUrl), query) {
					add = true
					break
				}
			}

			if add == false {
				if strings.Contains(strings.ToLower(tweet.User.Legacy.ScreenName), query) {
					add = true
				} else if strings.Contains(strings.ToLower(tweet.User.Legacy.Name), query) {
					add = true
				}
			}
		}

		if add {
			tweets = append(tweets, tweet)
		}
	}
	return tweets
}

func (a *Application) onNewTweet(ct *scraper.CachedTweet) bool {
	filename := path.Join(a.DataDir, ct.Tweet.IdStr+".json")
	if filesystem.Exist(filename) == false {
		conversation, err := a.Scraper.TweetDetail(ct.Tweet.IdStr)
		if err != nil {
			log.Error("Failed to fetch conversation %s: %s", ct.Tweet.IdStr, err.Error())
			return false
		}

		ct.Conversation = *conversation
		ct.Version = a.Build.Version

		d, err := json.Marshal(ct)
		if err == nil {
			err = ioutil.WriteFile(filename, d, 0644)
			if err == nil {
				a.tweets[ct.Tweet.IdStr] = ct

				ext, _ := GetFileExtensionFromUrl(ct.User.Legacy.ProfileImageUrlHttps)
				if ext == "" {
					ext = "blob"
				}
				userImageFilename := path.Join(a.DataDir, "media", ct.User.RestId+"."+ext)

				_ = a.Scraper.Download(ct.User.Legacy.ProfileImageUrlHttps, userImageFilename)

				for _, tweet := range conversation.GlobalObjects.Tweets {
					for _, ctm := range tweet.ExtendedEntities.Media {
						ext, _ = GetFileExtensionFromUrl(ctm.MediaUrlHttps)
						if ext == "" {
							ext = "blob"
						}
						mediaImageFilename := path.Join(a.DataDir, "media", ctm.IdStr+"."+ext)

						_ = a.Scraper.Download(ctm.MediaUrlHttps, mediaImageFilename)

						if ctm.Type == "video" {
							maxBitrate := 0
							videoUrl := ""
							for _, variant := range ctm.VideoInfo.Variants {
								if variant.Bitrate > maxBitrate {
									videoUrl = strings.TrimSuffix(variant.Url, "?tag=10")
									maxBitrate = variant.Bitrate
								}
							}

							if videoUrl != "" {
								ext, _ = GetFileExtensionFromUrl(videoUrl)
								if ext == "" {
									ext = "blob"
								}
								mediaVideoFilename := path.Join(a.DataDir, "media", ctm.IdStr+"."+ext)

								_ = a.Scraper.Download(videoUrl, mediaVideoFilename)
							}
						}
					}
				}

				r := NewResponse()
				r.Data["user"] = ct.User
				r.Data["tweet"] = ct.Tweet
				r.Data["conversation"] = ct.Conversation

				if b, e := r.Encode(); e == nil {
					a.Server.Hub().Broadcast(b)
				} else {
					log.Error("Failed to encode response: %s", e.Error())
					return false
				}
			}
		}

		if err != nil {
			log.Error("Failed to save tweet data: %s", err.Error())
			return false
		} else {
			log.Success("New tweet fetched: %s posted on %s", ct.Tweet.IdStr, ct.Tweet.CreatedAt)

			if a.Danger.RemoveBookmarks {
				r, err := a.Scraper.DeleteBookmarkDetail(ct.Tweet.IdStr)
				if err != nil {
					log.Error("Failed to remove remote bookmark %s: %s", ct.Tweet.IdStr, err.Error())
				} else if r.Data.TweetBookmarkDelete != "Done" {
					log.Info("Bookmark %s was already removed", ct.Tweet.IdStr)
				} else {
					log.Success("Bookmark removed: %s posted on %s", ct.Tweet.IdStr, ct.Tweet.CreatedAt)
				}
			}
		}
	} else {
		//log.Info("Tweet skipped (already fetched): %s posted on %s", ct.Tweet.IdStr, ct.Tweet.CreatedAt)
	}

	return true
}

func (a *Application) GetTweets() map[string]*scraper.CachedTweet {
	a.mx.RLock()
	defer a.mx.RUnlock()

	return a.tweets
}

func (a *Application) AddTweet(ct *scraper.CachedTweet) {
	a.mx.RLock()
	defer a.mx.RUnlock()

	a.tweets[ct.Tweet.IdStr] = ct
}

func (a *Application) SetState(state map[string]interface{}) {
	a.mx.Lock()
	defer a.mx.Unlock()

	a.state = state
}

func (a *Application) AddState(key string, value interface{}) {
	a.mx.Lock()
	defer a.mx.Unlock()

	a.state[key] = value
}

func (a *Application) GetState() map[string]interface{} {
	a.mx.RLock()
	defer a.mx.RUnlock()

	return a.state
}

func (a *Application) FormatTime(t interface{}) string {
	switch t.(type) {
	case time.Time:
		return t.(time.Time).Format("02.01.2006 15:04")
	case string:
		nt, _ := time.Parse("Mon Jan 02 15:04:05 -0700 2006", t.(string))
		return a.FormatTime(nt)
	}
	return t.(string)
}

func (a *Application) renderHtml(str string) template.HTML {
	return template.HTML(str)
}

func GetFileExtensionFromUrl(rawUrl string) (string, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return "", err
	}
	pos := strings.LastIndex(u.Path, ".")
	if pos == -1 {
		return "", errors.New("couldn't find a period to indicate a file extension")
	}
	return u.Path[pos+1 : len(u.Path)], nil
}
