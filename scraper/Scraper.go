package scraper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	FetchInterval = 60 * time.Second
)

type Scraper struct {
	AccessToken string `json:"access_token"`
	csrfToken   string
	Cookie      string `json:"cookie"`
	Section     string `json:"section"`
	variables   map[string]interface{}
	features    map[string]interface{}

	mx         sync.RWMutex
	close      chan bool
	OnNewTweet func(ct *CachedTweet) bool `json:"-"`
}

func NewScraper() *Scraper {
	return &Scraper{
		AccessToken: "",
		csrfToken:   "",
		Cookie:      "",
		Section:     "BvX-1Exs_MDBeKAedv2T_w",
		variables: map[string]interface{}{
			"count":                       20,
			"cursor":                      "",
			"includePromotedContent":      true,
			"withSuperFollowsUserFields":  true,
			"withDownvotePerspective":     false,
			"withReactionsMetadata":       false,
			"withReactionsPerspective":    false,
			"withSuperFollowsTweetFields": true,
		},
		features: map[string]interface{}{
			"dont_mention_me_view_api_enabled":      true,
			"interactive_text_enabled":              true,
			"responsive_web_uc_gql_enabled":         true,
			"vibe_api_enabled":                      true,
			"responsive_web_edit_tweet_api_enabled": false,
			"standardized_nudges_misinfo":           true,
			"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": false,
			"responsive_web_enhance_cards_enabled":                                    false,
		},
	}
}

func (s *Scraper) Start() {
	s.LoadCsrfToken()
	s.close = make(chan bool)

	ticker := time.NewTicker(FetchInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.run()
			case <-s.close:
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *Scraper) Stop() {
	s.close <- true
}

func (s *Scraper) LoadCsrfToken() bool {
	for _, p := range strings.Split(s.Cookie, ";") {
		parts := strings.SplitN(p, "=", 2)
		if strings.Trim(parts[0], " ") == "ct0" {
			s.csrfToken = strings.Trim(parts[1], " ")
			return true
		}
	}
	return false
}

func (s *Scraper) SetAccessTokens(AccessToken, Cookie string) bool {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.AccessToken = AccessToken
	s.Cookie = Cookie
	return s.LoadCsrfToken()
}

func (s *Scraper) buildUrl() string {
	jvb, _ := json.Marshal(s.variables)
	fvb, _ := json.Marshal(s.features)

	return fmt.Sprintf(
		"https://twitter.com/i/api/graphql/%s/Bookmarks?variables=%s&features=%s",
		s.Section,
		url.QueryEscape(string(jvb)),
		url.QueryEscape(string(fvb)),
	)
}

func (s *Scraper) run() {
	s.mx.Lock()
	defer s.mx.Unlock()

	req, err := http.NewRequest("GET", s.buildUrl(), nil)
	if err != nil {
		fmt.Printf("client: error making http request: %s\n", err)
		return
	}
	req.Header.Set("Cookie", s.Cookie)
	req.Header.Set("authorization", "Bearer "+s.AccessToken)
	req.Header.Set("x-csrf-token", s.csrfToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("client: error making http request: %s\n", err)
		return
	}

	if res.StatusCode != 200 {
		fmt.Printf("twitter: failed to fetch response body: %d\n", res.StatusCode)
		return
	}

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		return
	}

	rb := &BookmarkResponse{}
	if err := json.Unmarshal(resBody, rb); err != nil {
		fmt.Printf("twitter: could not read response body: %s\n", err)
		return
	}

	cursor := ""
	for _, instruction := range rb.Data.BookmarkTimeline.Timeline.Instructions {
		for _, entry := range instruction.Entries {
			switch entry.Content.EntryType {
			case "TimelineTimelineItem":
				// Tweet
				if s.OnNewTweet(&CachedTweet{
					User:  entry.Content.ItemContent.TweetResults.Result.Core.UserResults.Result,
					Tweet: entry.Content.ItemContent.TweetResults.Result.Legacy,
				}) == false {
					return
				}
			case "TimelineTimelineCursor":
				//Cursor
				if entry.Content.CursorType == "Bottom" {
					cursor = entry.Content.Value
				}
			}
		}
	}

	if cursor != "" {
		s.variables["cursor"] = cursor
		go s.run()
	}
}
