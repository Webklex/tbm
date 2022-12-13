package scraper

import (
	"time"
)

type CachedTweet struct {
	Version      string               `json:"version"`
	User         UserResult           `json:"user"`
	Tweet        TweetResult          `json:"tweet"`
	Conversation ConversationResponse `json:"conversation"`

	createdAt time.Time
}

type ThreadItem struct {
	Tweet TweetResult
	User  ConversationUser
}

func (ct *CachedTweet) CreatedAt() time.Time {
	if ct.createdAt.IsZero() {
		ct.createdAt, _ = time.Parse("Mon Jan 02 15:04:05 -0700 2006", ct.Tweet.CreatedAt)
	}
	return ct.createdAt
}

func (ct *CachedTweet) Thread() map[string]*ThreadItem {
	thread := map[string]*ThreadItem{}
	for tweetId, tweet := range ct.Conversation.GlobalObjects.Tweets {
		user, ok := ct.Conversation.GlobalObjects.Users[tweet.UserIdStr]
		if !ok {
			user = ConversationUser{}
		}

		thread[tweetId] = &ThreadItem{
			Tweet: tweet,
			User:  user,
		}
	}

	return thread
}
