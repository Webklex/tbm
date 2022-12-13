package app

import (
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"net/http"
	"strconv"
	"strings"
	"tbm/scraper"
	"tbm/server/response"
	"tbm/utils/log"
	"time"
)

func (a *Application) tweetsView(resp *response.ViewResponse) {
	paginator, err := a.paginateTweets(resp.Request())
	if err != nil {
		resp.AddError(err)
		return
	}
	paginator.Path = "/"
	resp.SetData(map[string]interface{}{
		"State":     a.GetState(),
		"Title":     "TBM - Bookmarks",
		"Paginator": paginator,
	})
}

func (a *Application) tweetView(resp *response.ViewResponse) {
	tweets := a.GetTweets()
	if cache, ok := tweets[resp.Parameter().ByName("id")]; ok {
		resp.SetData(map[string]interface{}{
			"State":  a.GetState(),
			"Title":  truncateTitle(bluemonday.StripTagsPolicy().Sanitize(cache.Tweet.FullText)),
			"Thread": cache.Thread(),
			"Tweet":  cache.Tweet,
			"User":   cache.User,
		})
		return
	}
	resp.AddError(response.NewErrorFromStatus(http.StatusNotFound))

	return
}

func (a *Application) configView(resp *response.ViewResponse) {
	resp.SetData(map[string]interface{}{
		"Title": "TBM - Config",
		"Config": map[string]interface{}{
			"ConfigFileName": a.ConfigFileName,
			"DataDir":        a.DataDir,
			"Mode":           a.Mode,
			"Danger":         a.Danger,
			"Host":           a.Server.Host,
			"Port":           a.Server.Port,
			"Delay":          a.Scraper.RawDelay,
			"Timeout":        a.Scraper.RawTimeout,
			"AccessToken":    a.Scraper.AccessToken,
			"Cookie":         a.Scraper.Cookie,
			"Index":          a.Scraper.Sections.Index,
			"Remove":         a.Scraper.Sections.Remove,
		},
	})
}

func (a *Application) updateConfigView(resp *response.ViewResponse) {
	resp.AddError(response.NewErrorFromString("Currently not supported", http.StatusNotImplemented))
	return

	if err := resp.Request().ParseForm(); err != nil {
		resp.AddError(response.NewError(err, http.StatusBadRequest))
		return
	}

	dataDir := resp.Request().FormValue("data_dir")
	mode := resp.Request().FormValue("mode")
	dangerRemoveBookmarks := resp.Request().FormValue("danger_remove_bookmarks")
	host := resp.Request().FormValue("host")
	port := resp.Request().FormValue("port")
	delay := resp.Request().FormValue("delay")
	timeout := resp.Request().FormValue("timeout")
	accessToken := resp.Request().FormValue("access_token")
	cookie := resp.Request().FormValue("cookie")
	index := resp.Request().FormValue("index")
	remove := resp.Request().FormValue("remove")

	go func() {
		log.Warning("Going to restart in 3 seconds...")
		time.Sleep(time.Second * 3)
		if err := a.Stop(); err != nil {
			log.Warning("failed to stop: %s", err.Error())
			return
		}

		// Update all configs
		//@TODO: Scraper doesn't stop correctly
		a.DataDir = dataDir
		a.Mode = ApplicationMode(mode)
		a.Danger.RemoveBookmarks = dangerRemoveBookmarks != ""

		a.Server.Host = host
		_port, _ := strconv.Atoi(port)
		a.Server.Port = uint(_port)

		a.Scraper.Timeout, _ = time.ParseDuration(timeout)
		a.Scraper.Delay, _ = time.ParseDuration(delay)
		a.Scraper.Cookie = cookie
		a.Scraper.AccessToken = accessToken
		a.Scraper.Sections.Index = index
		a.Scraper.Sections.Remove = remove

		fmt.Println("data_dir", dataDir)
		fmt.Println("mode", mode)
		fmt.Println("danger_remove_bookmarks", dangerRemoveBookmarks)
		fmt.Println("host", host)
		fmt.Println("port", port)
		fmt.Println("delay", delay)
		fmt.Println("timeout", timeout)
		fmt.Println("access_token", accessToken)
		fmt.Println("cookie", cookie)
		fmt.Println("index", index)
		fmt.Println("remove", remove)

		if err := a.Start(); err != nil {
			log.Warning("failed to start: %s", err.Error())
			return
		}
	}()

	a.configView(resp)
}

func (a *Application) statusView(resp *response.ViewResponse) {
	newest := time.Time{}
	oldest := time.Time{}
	for _, ct := range a.GetTweets() {
		createdAt := ct.CreatedAt()
		if newest.IsZero() || newest.Before(createdAt) {
			newest = createdAt
		}
		if oldest.IsZero() || oldest.After(createdAt) {
			oldest = createdAt
		}
	}
	resp.SetData(map[string]interface{}{
		"Cursor":         a.Scraper.GetCursor(),
		"TotalBookmarks": len(a.GetTweets()),
		"Build":          a.Build,
		"NewestTweet":    newest,
		"OldestTweet":    oldest,
		"State":          a.GetState(),
		"Scraper":        a.Scraper.IsRunning(),
		"Title":          "TBM - Status",
	})
}

func (a *Application) paginateTweets(req *http.Request) (*Paginator, *response.Error) {
	limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
	page, _ := strconv.Atoi(req.URL.Query().Get("page"))

	sortBy := req.URL.Query().Get("sort_by")
	order := req.URL.Query().Get("order")
	query := req.URL.Query().Get("query")

	tweets := make([]*scraper.CachedTweet, 0)
	if query != "" {
		tweets = a.SearchTweets(query)
	} else {
		for _, ct := range a.GetTweets() {
			tweets = append(tweets, ct)
		}
	}

	data := make([]interface{}, len(tweets))
	for i, v := range tweets {
		data[i] = v
	}

	paginator := NewPaginator(limit, page)
	paginator.SetData(data)
	paginator.Parameters["sort_by"] = sortBy
	paginator.Parameters["order"] = order
	paginator.Parameters["query"] = query

	if paginator.TotalPages < paginator.Page {
		return paginator, response.NewErrorFromStatus(http.StatusNotFound)
	}

	if strings.ToLower(order) != "desc" {
		order = "asc"
	}
	paginator.Sort(func(i, j interface{}) bool {
		tweet1 := i.(*scraper.CachedTweet)
		tweet2 := j.(*scraper.CachedTweet)

		switch strings.ToLower(sortBy) {
		case "quote_count", "quotecount", "quote":
			if order == "asc" {
				return tweet1.Tweet.QuoteCount < tweet2.Tweet.QuoteCount
			} else {
				return tweet1.Tweet.QuoteCount > tweet2.Tweet.QuoteCount
			}
		case "reply_count", "replycount", "reply":
			if order == "asc" {
				return tweet1.Tweet.ReplyCount < tweet2.Tweet.ReplyCount
			} else {
				return tweet1.Tweet.ReplyCount > tweet2.Tweet.ReplyCount
			}
		case "retweet_count", "retweetcount", "retweet":
			if order == "asc" {
				return tweet1.Tweet.RetweetCount < tweet2.Tweet.RetweetCount
			} else {
				return tweet1.Tweet.RetweetCount > tweet2.Tweet.RetweetCount
			}
		case "created_at", "createdat":
			if order == "asc" {
				return tweet1.CreatedAt().After(tweet2.CreatedAt())
			} else {
				return tweet1.CreatedAt().Before(tweet2.CreatedAt())
			}
		}
		return tweet1.CreatedAt().After(tweet2.CreatedAt())
	})

	return paginator, nil
}

func truncateTitle(title string, length ...int) string {
	if len(length) == 0 {
		length = []int{16}
	}
	if len(title) > length[0] {
		title = title[0:13] + "..."
	}
	return title
}
