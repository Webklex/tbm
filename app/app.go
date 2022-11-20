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
	Timezone string          `json:"timezone"`
	DataDir  string          `json:"data_dir"`
	Mode     ApplicationMode `json:"mode"`

	Build          Build  `json:"-"`
	ConfigFileName string `json:"-"`

	Server  *server.Server   `json:"server"`
	Scraper *scraper.Scraper `json:"scraper"`

	tweets        []*scraper.CachedTweet
	bookmarkIndex int
}

type Build struct {
	Number  string `json:"number"`
	Version string `json:"version"`
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
		Timezone:       "UTC",
		DataDir:        path.Join(dir, "data"),
		ConfigFileName: path.Join(dir, "config", "config.json"),
		Scraper:        scraper.NewScraper(),
		tweets:         make([]*scraper.CachedTweet, 0),
		bookmarkIndex:  1000000,
		Mode:           OnlineMode,
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
					if ct.Index != 0 && a.bookmarkIndex > ct.Index {
						a.bookmarkIndex = ct.Index
					} else if ct.Index == 0 {
						ct.Index = a.bookmarkIndex - 1
						a.bookmarkIndex = ct.Index
					}
					tweets = append(tweets, ct)
				}
			}
		}
	}
	sort.Slice(tweets, func(i, j int) bool {
		// t1, _ := time.Parse("Mon Jan 02 03:04:05 -0700 2006", tweets[i].Tweet.CreatedAt)
		// t2, _ := time.Parse("Mon Jan 02 03:04:05 -0700 2006", tweets[j].Tweet.CreatedAt)
		// return t1.Before(t2)
		return tweets[i].Index < tweets[j].Index
	})
	a.tweets = tweets
}

func (a *Application) Start() error {
	a.Server.AddState("mode", a.Mode)

	if a.Mode == OnlineMode {
		a.Scraper.Start()
	}
	return a.Server.Start()
}

func (a *Application) websocketCallback(m *server.Message) {
	t := &Task{}
	r := NewResponse()
	if err := json.Unmarshal(m.Content, t); err != nil {
		r.SetErrorStr("failed to decode message")
	}

	switch t.Command {
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
		query := strings.ToLower(_query.(string))
		tweets := make([]*scraper.CachedTweet, 0)
		for _, tweet := range a.tweets {
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
		r.Data["tweets"] = tweets
	} else {
		r.SetErrorStr("query parameter not found")
	}
}

func (a *Application) onNewTweet(ct *scraper.CachedTweet) bool {
	if ct.Tweet.IdStr == "" {
		fmt.Printf("empty tweet id. Probably got deleted at some point\n")
		return true
	}
	filename := path.Join(a.DataDir, ct.Tweet.IdStr+".json")
	if filesystem.Exist(filename) == false {
		conversation, err := a.Scraper.TweetDetail(ct.Tweet.IdStr)
		if err != nil {
			return false
		}

		a.bookmarkIndex--
		ct.Conversation = *conversation
		ct.Index = a.bookmarkIndex

		d, err := json.Marshal(ct)
		if err == nil {
			err = ioutil.WriteFile(filename, d, 0644)
			if err == nil {
				a.tweets = append(a.tweets, ct)

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
					fmt.Printf("failed to encode response: %s\n", e.Error())
					return false
				}
			}
		}

		if err != nil {
			fmt.Printf("Failed to save tweet data: %s\n", err.Error())
			return false
		} else {
			fmt.Printf("New tweet fetched: %s posted on %s\n", ct.Tweet.IdStr, ct.Tweet.CreatedAt)
		}
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
