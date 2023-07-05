package scraper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"tbm/utils/log"
	"time"
)

const (
	FetchInterval  = 1 * time.Minute
	CursorFilename = ".cursor.tmp"
)

type Scraper struct {
	AccessToken string `json:"access_token"`
	csrfToken   string
	Cookie      string   `json:"cookie"`
	Sections    Sections `json:"sections"`
	variables   map[string]interface{}
	features    map[string]interface{}
	cursor      string
	jsAppendix  string

	mx         sync.RWMutex
	close      chan bool
	running    bool
	onNewTweet OnNewTweetFunc

	Delay       time.Duration `json:"-"`
	Timeout     time.Duration `json:"-"`
	lastRequest time.Time

	RawTimeout string `json:"timeout"`
	RawDelay   string `json:"delay"`
}

type Sections struct {
	Index  string `json:"index"`
	Remove string `json:"remove"`
	Detail string `json:"detail"`
}

type OnNewTweetFunc func(ct *CachedTweet) bool

func NewScraper(onNewTweet OnNewTweetFunc) *Scraper {
	return &Scraper{
		AccessToken: "",
		csrfToken:   "",
		Cookie:      "",
		cursor:      "",
		jsAppendix:  "",
		running:     false,
		onNewTweet:  onNewTweet,
		Sections: Sections{
			Index:  "",
			Remove: "",
		},
		Delay:       time.Second * 30,
		Timeout:     time.Second * 10,
		lastRequest: time.Time{},
		variables: map[string]interface{}{
			"count":                  20,
			"cursor":                 "",
			"includePromotedContent": false,
			//"withSuperFollowsUserFields":  true,
			//"withDownvotePerspective":     false,
			//"withReactionsMetadata":       false,
			//"withReactionsPerspective":    false,
			//"withSuperFollowsTweetFields": true,
		},
		features: map[string]interface{}{
			"graphql_timeline_v2_bookmark_timeline":                                   true,
			"rweb_lists_timeline_redesign_enabled":                                    true,
			"responsive_web_graphql_exclude_directive_enabled":                        true,
			"verified_phone_label_enabled":                                            false,
			"creator_subscriptions_tweet_preview_api_enabled":                         true,
			"responsive_web_graphql_timeline_navigation_enabled":                      true,
			"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
			"tweetypie_unmention_optimization_enabled":                                true,
			"responsive_web_edit_tweet_api_enabled":                                   true,
			"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
			"view_counts_everywhere_api_enabled":                                      true,
			"longform_notetweets_consumption_enabled":                                 true,
			"responsive_web_twitter_article_tweet_consumption_enabled":                false,
			"tweet_awards_web_tipping_enabled":                                        false,
			"freedom_of_speech_not_reach_fetch_enabled":                               true,
			"standardized_nudges_misinfo":                                             true,
			"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
			"longform_notetweets_rich_text_read_enabled":                              true,
			"longform_notetweets_inline_media_enabled":                                true,
			"responsive_web_media_download_video_enabled":                             false,
			"responsive_web_enhance_cards_enabled":                                    false,
		},
	}
}

func (s *Scraper) Start(removeBookmarks bool) {
	s.LoadCsrfToken()
	if s.Sections.Index == "" || s.Sections.Remove == "" || s.AccessToken == "" {
		if err := s.LoadSections(); err != nil {
			log.Error(err)
			return
		}
	}
	log.Info("Scraper started")

	if b, err := os.ReadFile(CursorFilename); err == nil {
		s.cursor = string(b)
	}

	go s.Run(removeBookmarks)
	s.close = make(chan bool)

	ticker := time.NewTicker(FetchInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.Run(removeBookmarks)
			case <-s.close:
				ticker.Stop()
				s.close = nil
				log.Warning("Scraper stopped")
				return
			}
		}
	}()
}

func (s *Scraper) Stop() {
	if s.close != nil {
		s.close <- true
	}
}

func (s *Scraper) delayRequest() {
	if s.lastRequest.IsZero() {
		return
	}
	delta := s.Delay - time.Now().Sub(s.lastRequest)
	if delta > 0 {
		time.Sleep(delta)
	}
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

func (s *Scraper) LoadSections() error {
	src := "https://twitter.com/i/bookmarks"
	req, err := http.NewRequest("GET", src, nil)
	if err != nil {
		return err
	}
	req.Header.Set("cookie", s.Cookie)

	// Something similar might be required in the future, if the legacy version gets removed
	//req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/113.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("failed to download resource with \"" + resp.Status + "\" from " + src)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	bookmarkHtml := string(b)

	// Check if the legacy version is used
	isLegacy := strings.Contains(bookmarkHtml, "https://abs.twimg.com/responsive-web/client-web-legacy/main")

	if isLegacy {
		if err := s.fetchMainJs(bookmarkHtml, `https://abs\.twimg\.com/responsive-web/client-web-legacy/main\.([0-9a-zA-Z]*)\.js`); err != nil {
			return err
		}
	} else if err := s.fetchMainJs(bookmarkHtml, `https://abs\.twimg\.com/responsive-web/client-web/main\.([0-9a-zA-Z]*)\.js`); err != nil {
		return err
	}

	re := regexp.MustCompile(`",api:"([0-9a-zA-Z]*)",`)
	matches := re.FindStringSubmatch(bookmarkHtml)
	if len(matches) > 0 {
		apiJsFileToken := matches[1] + s.jsAppendix

		if isLegacy {
			if err := s.fetchApiJs("https://abs.twimg.com/responsive-web/client-web-legacy/api." + apiJsFileToken + ".js"); err != nil {
				return err
			}
		} else if err := s.fetchApiJs("https://abs.twimg.com/responsive-web/client-web/api." + apiJsFileToken + ".js"); err != nil {
			return err
		}
	} else {
		return errors.New("failed to locate bookmark index section file")
	}

	return nil
}

func (s *Scraper) fetchApiJs(apiJsFile string) error {
	req, err := http.NewRequest("GET", apiJsFile, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("failed to download resource with \"" + resp.Status + "\" from " + apiJsFile)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	jsContent := string(b)
	re := regexp.MustCompile(`"([a-zA-Z0-9-_]*)",operationName:"Bookmarks"`)
	matches := re.FindStringSubmatch(jsContent)
	if len(matches) > 1 {
		if s.Sections.Index == "" {
			s.Sections.Index = matches[1]
		}
	} else {
		return errors.New("failed to locate bookmark index section")
	}

	re = regexp.MustCompile(`"([a-zA-Z0-9-_]*)",operationName:"TweetDetail"`)
	matches = re.FindStringSubmatch(jsContent)
	if len(matches) > 1 {
		if s.Sections.Detail == "" {
			s.Sections.Detail = matches[1]
		}
	} else {
		return errors.New("failed to locate bookmark detail section")
	}

	return nil
}

func (s *Scraper) fetchMainJs(bookmarkHtml string, regex string) error {

	re := regexp.MustCompile(regex)
	matches := re.FindStringSubmatch(bookmarkHtml)
	if len(matches) > 0 {
		mainJsFile := matches[0]

		req, err := http.NewRequest("GET", mainJsFile, nil)
		if err != nil {
			return err
		}

		resp, err := http.DefaultClient.Do(req)

		if err != nil {
			return err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return errors.New("failed to download resource with \"" + resp.Status + "\" from " + mainJsFile)
		}

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		jsContent := string(b)

		re = regexp.MustCompile(`"([a-zA-Z0-9-_]*)",operationName:"DeleteBookmark"`)
		matches = re.FindStringSubmatch(jsContent)

		if len(matches) > 1 {
			if s.Sections.Remove == "" {
				s.Sections.Remove = matches[1]
			}
		} else {
			return errors.New("failed to locate bookmark remove section")
		}

		re = regexp.MustCompile(`AAAAAAAAAAAAAAA([a-zA-Z0-9-_%]*)`)
		matches = re.FindStringSubmatch(jsContent)
		if len(matches) > 1 {
			if s.AccessToken == "" {
				s.AccessToken = matches[0]
			}
		} else {
			return errors.New("failed to locate the access token")
		}

		// Get the last char from the js file path excluding the extension
		re = regexp.MustCompile(`main\.([0-9a-zA-Z]*)\.js`)
		matches = re.FindStringSubmatch(mainJsFile)
		if len(matches) > 1 {
			s.jsAppendix = matches[1][len(matches[1])-1:]
		} else {
			return errors.New("failed to locate the js appendix")
		}
	} else {
		return errors.New("failed to locate main js file")
	}
	return nil
}

func (s *Scraper) SetAccessTokens(AccessToken, Cookie string) bool {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.AccessToken = AccessToken
	s.Cookie = Cookie
	return s.LoadCsrfToken()
}

func (s *Scraper) buildUrl() string {
	s.variables["cursor"] = s.cursor
	jvb, _ := json.Marshal(s.variables)
	fvb, _ := json.Marshal(s.features)

	return fmt.Sprintf(
		"https://twitter.com/i/api/graphql/%s/Bookmarks?variables=%s&features=%s",
		s.Sections.Index,
		url.QueryEscape(string(jvb)),
		url.QueryEscape(string(fvb)),
	)
}

func (s *Scraper) IsRunning() bool {
	s.mx.RLock()
	defer s.mx.RUnlock()

	return s.running
}

func (s *Scraper) Run(keepCursor bool) {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.running {
		return
	}
	s.running = true
	go s.run(keepCursor)
}

func (s *Scraper) GetCursor() string {
	s.mx.RLock()
	defer s.mx.RUnlock()

	return s.cursor
}

func (s *Scraper) SetCursor(cursor string) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.setCursor(cursor)
}

func (s *Scraper) setCursor(cursor string) {
	s.cursor = cursor

	f, err := os.OpenFile(CursorFilename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, _ = f.WriteString(s.cursor)
}

func (s *Scraper) run(keepCursor bool, attempts ...error) {
	if len(attempts) > 10 {
		log.Error("Api failed to many times: %s", attempts[len(attempts)-1].Error())
		s.free()
		return
	}

	req, err := http.NewRequest("GET", s.buildUrl(), nil)
	if err != nil {
		log.Error("client: error making http request: %s", err.Error())
		s.free()
		return
	}
	req.Header.Set("Cookie", s.Cookie)
	req.Header.Set("authorization", "Bearer "+s.AccessToken)
	req.Header.Set("x-csrf-token", s.csrfToken)

	s.delayRequest()
	res, err := http.DefaultClient.Do(req)
	s.mx.Lock()
	s.lastRequest = time.Now()
	s.mx.Unlock()

	if err != nil {
		log.Error("client: error sending http request: %s", err.Error())
		s.free()
		return
	}

	if res.StatusCode != 200 {
		log.Error("twitter: failed to fetch response body: %s", res.Status)
		s.free()
		return
	}

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error("client: could not read response body: %s", err)
		s.free()
		return
	}

	rb := &BookmarkResponse{}
	if err := json.Unmarshal(resBody, rb); err != nil {
		log.Error("client: could not unmarshal response body: %s", err)
		s.free()
		return
	}

	if len(rb.Errors) > 0 {
		err := rb.Errors[0]
		log.Warning("twitter: api error at cursor \"%s\" with %s", s.cursor, err.Message)

		attempts = append(attempts, errors.New(err.Message))
		go s.run(keepCursor, attempts...)
		return
	}

	cursor := ""
	count := 0
	empty := 0
	for _, instruction := range rb.Data.BookmarkTimeline.Timeline.Instructions {
		for _, entry := range instruction.Entries {
			switch entry.Content.EntryType {
			case "TimelineTimelineItem":
				// Tweet
				tweet := entry.Content.ItemContent.TweetResults.Result.Legacy
				user := entry.Content.ItemContent.TweetResults.Result.Core.UserResults.Result
				if tweet.IdStr == "" {
					tweet = entry.Content.ItemContent.TweetResults.Result.Tweet.Legacy
					user = entry.Content.ItemContent.TweetResults.Result.Tweet.Core.UserResults.Result
				}

				if tweet.IdStr == "" {
					log.Info("Empty tweet data. %s probably got deleted at some point", entry.EntryId)
					// @TODO: might want to call
					// 		  s.DeleteBookmarkDetail(entry.Content.ItemContent.TweetResults.Result.RestId)
					//		  to delete this bookmark - but it might also be a twitter issue and the tweet becomes
					//		  available at a later point. I'm assuming RestId equals IdStr, but I could be wrong..
					empty++
				} else {
					if s.onNewTweet(&CachedTweet{
						User:  user,
						Tweet: tweet,
					}) == false {
						go s.run(keepCursor, attempts...)
						return
					}
				}
				count++
			case "TimelineTimelineCursor":
				//Cursor
				if entry.Content.CursorType == "Bottom" {
					cursor = entry.Content.Value
				}
			}
		}
	}

	s.mx.Lock()
	defer s.mx.Unlock()

	if keepCursor {
		if c, ok := s.variables["count"].(int); ok && c <= count {
			if count == empty {
				s.setCursor(cursor)
			}
			go s.run(keepCursor)
		} else {
			s.free()
		}
	} else if cursor != "" {
		s.setCursor(cursor)
		go s.run(keepCursor)
	} else {
		s.free()
	}
}

func (s *Scraper) free() {
	go func() {
		s.mx.Lock()
		defer s.mx.Unlock()
		s.running = false
	}()
}

func (s *Scraper) Download(src, target string) error {
	b, err := s.Get(src)
	if err != nil {
		log.Error("twitter: could not read response body: %s", err)
		return err
	}

	return os.WriteFile(target, b, 0644)
}

type RemoveBookmarkResponse struct {
	Data struct {
		TweetBookmarkDelete string `json:"tweet_bookmark_delete"`
	} `json:"data"`
}

func (s *Scraper) DeleteBookmarkDetail(id string) (*RemoveBookmarkResponse, error) {
	b, err := json.Marshal(map[string]interface{}{
		"variables": map[string]string{
			"tweet_id": id,
		},
		"queryId": s.Sections.Remove,
	})

	req, err := http.NewRequest("POST", "https://twitter.com/i/api/graphql/"+s.Sections.Remove+"/DeleteBookmark", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", s.Cookie)
	req.Header.Set("authorization", "Bearer "+s.AccessToken)
	req.Header.Set("x-csrf-token", s.csrfToken)
	req.Header.Set("content-type", "application/json")

	s.delayRequest()
	resp, err := http.DefaultClient.Do(req)
	s.lastRequest = time.Now()

	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("failed to remove bookmark " + id + " with status \"" + resp.Status)
	}

	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	v := &RemoveBookmarkResponse{}
	if err := json.Unmarshal(rb, v); err != nil {
		return nil, err
	}

	return v, nil
}

func (s *Scraper) TweetDetail(id string) (*ConversationResponse, error) {
	variables, err := json.Marshal(map[string]interface{}{
		//"cursor":                               "",
		"focalTweetId":                           id,
		"referrer":                               "bookmarks",
		"with_rux_injections":                    false,
		"includePromotedContent":                 true,
		"withCommunity":                          true,
		"withQuickPromoteEligibilityTweetFields": true,
		"withBirdwatchNotes":                     true,
		"withVoice":                              true,
		"withV2Timeline":                         true,
	})
	if err != nil {
		return nil, err
	}

	features, err := json.Marshal(map[string]interface{}{
		"rweb_lists_timeline_redesign_enabled":                                    true,
		"responsive_web_graphql_exclude_directive_enabled":                        true,
		"verified_phone_label_enabled":                                            false,
		"creator_subscriptions_tweet_preview_api_enabled":                         true,
		"responsive_web_graphql_timeline_navigation_enabled":                      true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
		"tweetypie_unmention_optimization_enabled":                                true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
		"view_counts_everywhere_api_enabled":                                      true,
		"longform_notetweets_consumption_enabled":                                 true,
		"responsive_web_twitter_article_tweet_consumption_enabled":                false,
		"tweet_awards_web_tipping_enabled":                                        false,
		"freedom_of_speech_not_reach_fetch_enabled":                               true,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
		"longform_notetweets_rich_text_read_enabled":                              true,
		"longform_notetweets_inline_media_enabled":                                true,
		"responsive_web_media_download_video_enabled":                             false,
		"responsive_web_enhance_cards_enabled":                                    false,
	})
	if err != nil {
		return nil, err
	}

	fieldToggles, err := json.Marshal(map[string]interface{}{
		"withArticleRichContentState": false,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", "https://twitter.com/i/api/graphql/"+s.Sections.Detail+"/TweetDetail?variables="+
		url.QueryEscape(string(variables))+"&features="+
		url.QueryEscape(string(features))+"&fieldToggles="+
		url.QueryEscape(string(fieldToggles)), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", s.Cookie)
	req.Header.Set("authorization", "Bearer "+s.AccessToken)
	req.Header.Set("x-csrf-token", s.csrfToken)
	req.Header.Set("content-type", "application/json")

	s.delayRequest()
	resp, err := http.DefaultClient.Do(req)
	s.lastRequest = time.Now()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("failed to download resource with \"" + resp.Status + "\" from twitter.com")
	}

	// The following part won't work because the response structure has changed. Everything up to this point should work.
	// This call has to be repeated until all conversations have been fetched. This can be accomplished by providing the previously received cursor.
	// However, ConversationResponse has to be modified before continuing...
	fmt.Println("Please see: https://github.com/Webklex/tbm/issues/31")
	os.Exit(2)

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	v := &ConversationResponse{}
	if err := json.Unmarshal(b, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Scraper) Get(src string) ([]byte, error) {
	req, err := http.NewRequest("GET", src, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", s.Cookie)
	req.Header.Set("authorization", "Bearer "+s.AccessToken)
	req.Header.Set("x-csrf-token", s.csrfToken)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("failed to download resource with \"" + resp.Status + "\" from " + src)
	}

	return io.ReadAll(resp.Body)
}
