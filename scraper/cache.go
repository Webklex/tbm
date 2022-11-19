package scraper

type CachedTweet struct {
	Index        int                  `json:"index"`
	User         UserResult           `json:"user"`
	Tweet        TweetResult          `json:"tweet"`
	Conversation ConversationResponse `json:"conversation"`
}
