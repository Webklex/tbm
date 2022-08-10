package scraper

type CachedTweet struct {
	User  UserResult  `json:"user"`
	Tweet TweetResult `json:"tweet"`
}
