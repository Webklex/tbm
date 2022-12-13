package app

import (
	"net/http"
	"tbm/server/response"
	"time"
)

func (a *Application) stateEndpoint(resp *response.JsonResponse) {
	resp.SetData(a.GetState())
}

func (a *Application) tweetsEndpoint(resp *response.JsonResponse) {
	paginator, err := a.paginateTweets(resp.Request())
	if err != nil {
		resp.AddError(err)
		return
	}
	resp.SetData(map[string]interface{}{
		"page":       paginator.Page,
		"limit":      paginator.Limit,
		"Total":      paginator.Total,
		"TotalPages": paginator.TotalPages,
		"Data":       paginator.Data(),
		"Links":      paginator.Links(5),
	})
}

func (a *Application) tweetEndpoint(resp *response.JsonResponse) {
	tweets := a.GetTweets()
	if cache, ok := tweets[resp.Parameter().ByName("id")]; ok {
		resp.SetData(map[string]interface{}{
			"Thread": cache.Thread(),
			"Tweet":  cache.Tweet,
			"User":   cache.User,
		})
		return
	}
	resp.AddError(response.NewErrorFromStatus(http.StatusNotFound))

	return
}

func (a *Application) statusEndpoint(resp *response.JsonResponse) {
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
	})
}
