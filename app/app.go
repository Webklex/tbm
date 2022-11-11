package app

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"tbm/scraper"
	"tbm/server"
	"tbm/utils/filesystem"
	"time"
)

type Application struct {
	Timezone string `json:"timezone"`
	DataDir  string `json:"data_dir"`

	Build          Build  `json:"-"`
	ConfigFileName string `json:"-"`

	Server *server.Server `json:"server"`

	Scraper *scraper.Scraper `json:"scraper"`
	tweets  []*scraper.CachedTweet
}

type Build struct {
	Number  string `json:"number"`
	Version string `json:"version"`
}

func NewApplication(assets embed.FS) *Application {
	dir, _ := os.Getwd()

	a := &Application{
		Timezone:       "UTC",
		DataDir:        path.Join(dir, "data"),
		ConfigFileName: path.Join(dir, "config", "config.json"),
		Scraper:        scraper.NewScraper(),
		tweets:         make([]*scraper.CachedTweet, 0),
	}
	a.Server = server.NewServer(a.websocketCallback, assets)
	a.Scraper.OnNewTweet = a.onNewTweet

	return a
}

func (a *Application) loadConfigFile() error {
	if _, err := os.Stat(a.ConfigFileName); err == nil {
		content, err := ioutil.ReadFile(a.ConfigFileName)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(content, a); err != nil {
			return err
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
	a.Server.Load(path.Join(a.DataDir, "media"))
	a.LoadTweetCache()

	return nil
}

func (a *Application) LoadTweetCache() {
	items, _ := ioutil.ReadDir(a.DataDir)
	tweets := make([]*scraper.CachedTweet, 0)
	for _, item := range items {
		if item.IsDir() == false {
			dat, err := os.ReadFile(path.Join(a.DataDir, item.Name()))
			if err == nil {
				ct := &scraper.CachedTweet{}
				if err := json.Unmarshal(dat, ct); err == nil {
					tweets = append(tweets, ct)
				}
			}
		}
	}
	sort.Slice(tweets, func(i, j int) bool {
		t1, _ := time.Parse("Mon Jan 02 03:04:05 -0700 2006", tweets[i].Tweet.CreatedAt)
		t2, _ := time.Parse("Mon Jan 02 03:04:05 -0700 2006", tweets[j].Tweet.CreatedAt)
		return t1.Before(t2)
	})
	a.tweets = tweets
}

func (a *Application) Start() error {
	a.Scraper.Start()
	return a.Server.Start()
}

func (a *Application) websocketCallback(m *server.Message) {
	t := &Task{}
	r := NewResponse()
	if err := json.Unmarshal(m.Content, t); err != nil {
		r.SetErrorStr("failed to decode message")
	}

	switch t.Command {
	case "set_tokens":
		a.setTokens(t, r)
	case "get_tweets":
		r.Data["tweets"] = a.tweets
	case "search_tweets":
		a.searchTweets(t, r)
	default:
		r.SetErrorStr("unknown command")
	}

	if b, err := r.Encode(); err == nil {
		m.Client.Send(b)
	} else {
		fmt.Printf("failed to encode response: %s\n", err.Error())
	}
}

func (a *Application) searchTweets(t *Task, r *Response) {
	if _query, ok := t.Payload["query"]; ok {
		query := _query.(string)
		tweets := make([]*scraper.CachedTweet, 0)
		for _, tweet := range a.tweets {
			add := strings.Contains(tweet.Tweet.FullText, query)
			if add == false {
				for _, u := range tweet.Tweet.Entities.Urls {
					if strings.Contains(u.ExpandedUrl, query) {
						add = true
						break
					}
				}

				if add == false {
					if strings.Contains(tweet.User.Legacy.ScreenName, query) {
						add = true
					} else if strings.Contains(tweet.User.Legacy.Name, query) {
						add = true
					}
				}
			}

			if add {
				tweets = append(tweets, tweet)
			}
		}
		r.Data["tweets"] = tweets
	} else {
		r.SetErrorStr("query parameter not found")
	}
}

func (a *Application) setTokens(t *Task, r *Response) {
	if accessToken, ok := t.Payload["access_token"]; ok && accessToken != "" {
		if cookie, ok := t.Payload["cookie"]; ok && cookie != "" {
			if a.Scraper.SetAccessTokens(accessToken.(string), cookie.(string)) == false {
				r.SetErrorStr("csrf token could not be found inside the cookie")
			} else {
				r.Data["status"] = "OK"
			}
		} else {
			r.SetErrorStr("cookie not found")
		}
	} else {
		r.SetErrorStr("access token not found")
	}
}

func (a *Application) onNewTweet(ct *scraper.CachedTweet) bool {
	if ct.Tweet.IdStr == "" {
		fmt.Printf("empty tweet id. Probably got deleted at some point\n")
		return true
	}
	d, err := json.Marshal(ct)
	if err == nil {
		filename := path.Join(a.DataDir, ct.Tweet.IdStr+".json")
		if filesystem.Exist(filename) == false {
			err = ioutil.WriteFile(filename, d, 0644)
			if err == nil {
				a.tweets = append(a.tweets, ct)

				r := NewResponse()
				r.Data["user"] = ct.User
				r.Data["tweet"] = ct.Tweet

				ext, _ := GetFileExtensionFromUrl(ct.User.Legacy.ProfileImageUrlHttps)
				if ext == "" {
					ext = "blob"
				}
				userImageFilename := path.Join(a.DataDir, "media", ct.User.RestId+"."+ext)

				_ = a.Scraper.Download(ct.User.Legacy.ProfileImageUrlHttps, userImageFilename)
				if ct.Tweet.Entities.Media != nil {
					for _, media := range ct.Tweet.Entities.Media {
						ext, _ = GetFileExtensionFromUrl(media.MediaUrlHttps)
						if ext == "" {
							ext = "blob"
						}
						mediaImageFilename := path.Join(a.DataDir, "media", media.IdStr+"."+ext)

						_ = a.Scraper.Download(media.MediaUrlHttps, mediaImageFilename)
					}
				}

				if b, e := r.Encode(); e == nil {
					a.Server.Hub().Broadcast(b)
				} else {
					fmt.Printf("failed to encode response: %s\n", e.Error())
					return false
				}
			}
		} else {
			return false
		}
	}

	if err != nil {
		fmt.Printf("Failed to save tweet data: %s\n", err.Error())
		return false
	} else {
		fmt.Printf("New tweet fetched: %s\n", ct.Tweet.IdStr)
	}
	return true
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
