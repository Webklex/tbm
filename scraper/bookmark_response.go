package scraper

import (
	"regexp"
	"strings"
	"time"
)

type BookmarkResponse struct {
	Data struct {
		BookmarkTimeline struct {
			Timeline struct {
				Instructions []struct {
					Type    string `json:"type"`
					Entries []struct {
						EntryId   string `json:"entryId"`
						SortIndex string `json:"sortIndex"`
						Content   struct {
							EntryType   string `json:"entryType"`
							TypeName    string `json:"__typename"`
							ItemContent struct {
								ItemType            string `json:"itemType"`
								TypeName            string `json:"__typename"`
								Value               string `json:"value"`
								CursorType          string `json:"cursorType"`
								StopOnEmptyResponse bool   `json:"stopOnEmptyResponse"`
								TweetResults        struct {
									Result struct {
										TweetResultBlock
										Tweet TweetResultBlock `json:"tweet"`
									} `json:"result"`
								} `json:"tweet_results"`
								TweetDisplayType string `json:"tweetDisplayType"`
							} `json:"itemContent"`
							Value               string `json:"value"`
							CursorType          string `json:"cursorType"`
							StopOnEmptyResponse bool   `json:"stopOnEmptyResponse"`
						} `json:"content"`
					} `json:"entries"`
				} `json:"instructions"`
				ResponseObjects struct {
					FeedbackActions    []interface{} `json:"feedbackActions"`
					ImmediateReactions []interface{} `json:"immediateReactions"`
				} `json:"responseObjects"`
			} `json:"timeline"`
		} `json:"bookmark_timeline"`
	} `json:"data"`

	Errors []struct {
		Message   string `json:"message"`
		Locations []struct {
			Line   int `json:"line"`
			Column int `json:"column"`
		} `json:"locations"`
		Path       []string `json:"path"`
		Extensions struct {
			Name    string `json:"name"`
			Source  string `json:"source"`
			Code    int    `json:"code"`
			Kind    string `json:"kind"`
			Tracing struct {
				TraceId string `json:"trace_id"`
			} `json:"tracing"`
		} `json:"extensions"`
		Code    int    `json:"code"`
		Kind    string `json:"kind"`
		Name    string `json:"name"`
		Source  string `json:"source"`
		Tracing struct {
			TraceId string `json:"trace_id"`
		} `json:"tracing"`
	} `json:"errors"`
}

type TweetResultBlock struct {
	TypeName string `json:"__typename"`
	RestId   string `json:"rest_id"`
	Core     struct {
		UserResults struct {
			Result UserResult `json:"result"`
		} `json:"user_results"`
	} `json:"core"`
	UnmentionInfo interface{} `json:"unmention_info"`
	Legacy        TweetResult `json:"legacy"`
}

type UserResult struct {
	TypeName                   string      `json:"__typename"`
	Id                         string      `json:"id"`
	RestId                     string      `json:"rest_id"`
	AffiliatesHighlightedLabel interface{} `json:"affiliates_highlighted_label"`
	HasNftAvatar               bool        `json:"has_nft_avatar"`
	Legacy                     struct {
		BlockedBy           bool   `json:"blocked_by"`
		Blocking            bool   `json:"blocking"`
		CanDm               bool   `json:"can_dm"`
		CanMediaTag         bool   `json:"can_media_tag"`
		CreatedAt           string `json:"created_at"`
		DefaultProfile      bool   `json:"default_profile"`
		DefaultProfileImage bool   `json:"default_profile_image"`
		Description         string `json:"description"`
		Entities            struct {
			Description struct {
				Urls []struct {
					DisplayUrl  string `json:"display_url"`
					ExpandedUrl string `json:"expanded_url"`
					Url         string `json:"url"`
					Indices     []int  `json:"indices"`
				} `json:"urls"`
			} `json:"description"`
			Url struct {
				Urls []struct {
					DisplayUrl  string `json:"display_url"`
					ExpandedUrl string `json:"expanded_url"`
					Url         string `json:"url"`
					Indices     []int  `json:"indices"`
				} `json:"urls"`
			} `json:"url"`
		} `json:"entities"`
		FastFollowersCount      int      `json:"fast_followers_count"`
		FavouritesCount         int      `json:"favourites_count"`
		FollowRequestSent       bool     `json:"follow_request_sent"`
		FollowedBy              bool     `json:"followed_by"`
		FollowersCount          int      `json:"followers_count"`
		Following               bool     `json:"following"`
		FriendsCount            int      `json:"friends_count"`
		HasCustomTimelines      bool     `json:"has_custom_timelines"`
		IsTranslator            bool     `json:"is_translator"`
		ListedCount             int      `json:"listed_count"`
		Location                string   `json:"location"`
		MediaCount              int      `json:"media_count"`
		Muting                  bool     `json:"muting"`
		Name                    string   `json:"name"`
		NormalFollowersCount    int      `json:"normal_followers_count"`
		Notifications           bool     `json:"notifications"`
		PinnedTweetIdsStr       []string `json:"pinned_tweet_ids_str"`
		PossiblySensitive       bool     `json:"possibly_sensitive"`
		ProfileBannerExtensions struct {
			MediaColor struct {
				R struct {
					Ok struct {
						Palette []struct {
							Percentage float64 `json:"percentage"`
							Rgb        struct {
								Blue  int `json:"blue"`
								Green int `json:"green"`
								Red   int `json:"red"`
							} `json:"rgb"`
						} `json:"palette"`
					} `json:"ok"`
				} `json:"r"`
			} `json:"mediaColor"`
		} `json:"profile_banner_extensions"`
		ProfileBannerUrl       string `json:"profile_banner_url"`
		ProfileImageExtensions struct {
			MediaColor struct {
				R struct {
					Ok struct {
						Palette []struct {
							Percentage float64 `json:"percentage"`
							Rgb        struct {
								Blue  int `json:"blue"`
								Green int `json:"green"`
								Red   int `json:"red"`
							} `json:"rgb"`
						} `json:"palette"`
					} `json:"ok"`
				} `json:"r"`
			} `json:"mediaColor"`
		} `json:"profile_image_extensions"`
		ProfileImageUrlHttps    string        `json:"profile_image_url_https"`
		ProfileInterstitialType string        `json:"profile_interstitial_type"`
		Protected               bool          `json:"protected"`
		ScreenName              string        `json:"screen_name"`
		StatusesCount           int           `json:"statuses_count"`
		TranslatorType          string        `json:"translator_type"`
		Url                     string        `json:"url"`
		Verified                bool          `json:"verified"`
		WantRetweets            bool          `json:"want_retweets"`
		WithheldInCountries     []interface{} `json:"withheld_in_countries"`
	} `json:"legacy"`
	Professional struct {
		RestId           string        `json:"rest_id"`
		ProfessionalType string        `json:"professional_type"`
		Category         []interface{} `json:"category"`
	} `json:"professional"`
	SuperFollowEligible bool `json:"super_follow_eligible"`
	SuperFollowedBy     bool `json:"super_followed_by"`
	SuperFollowing      bool `json:"super_following"`
}

type TweetResult struct {
	CreatedAt            string    `json:"created_at"`
	InReplyToStatusIDStr string    `json:"in_reply_to_status_id_str"`
	Place                Place     `json:"place"`
	RetweetedStatusIDStr string    `json:"retweeted_status_id_str"`
	QuotedStatusIDStr    string    `json:"quoted_status_id_str"`
	Time                 time.Time `json:"time"`
	ConversationIdStr    string    `json:"conversation_id_str"`
	DisplayTextRange     []int     `json:"display_text_range"`
	Entities             struct {
		Media []struct {
			DisplayUrl    string `json:"display_url"`
			ExpandedUrl   string `json:"expanded_url"`
			IdStr         string `json:"id_str"`
			Indices       []int  `json:"indices"`
			MediaUrlHttps string `json:"media_url_https"`
			Url           string `json:"url"`
			Type          string `json:"type"`
			Features      struct {
				Large struct {
					Faces []interface{} `json:"faces"`
				} `json:"large"`
				Medium struct {
					Faces []interface{} `json:"medium"`
				} `json:"medium"`
				Small struct {
					Faces []interface{} `json:"small"`
				} `json:"small"`
				Orig struct {
					Faces []interface{} `json:"orig"`
				} `json:"orig"`
			} `json:"features"`
			Sizes struct {
				Large struct {
					H      int    `json:"h"`
					W      int    `json:"w"`
					Resize string `json:"resize"`
				} `json:"large"`
				Medium struct {
					H      int    `json:"h"`
					W      int    `json:"w"`
					Resize string `json:"resize"`
				} `json:"medium"`
				Small struct {
					H      int    `json:"h"`
					W      int    `json:"w"`
					Resize string `json:"resize"`
				} `json:"small"`
				Thumb struct {
					H      int    `json:"h"`
					W      int    `json:"w"`
					Resize string `json:"resize"`
				} `json:"thumb"`
			} `json:"sizes"`
			OriginalInfo struct {
				Height     int `json:"height"`
				Width      int `json:"width"`
				FocusRects []struct {
					X int `json:"x"`
					Y int `json:"y"`
					H int `json:"h"`
					W int `json:"w"`
				} `json:"focus_rects"`
			} `json:"original_info"`
		} `json:"media"`
		UserMentions []ConversationUser `json:"user_mentions"`
		Urls         []struct {
			DisplayUrl  string `json:"display_url"`
			ExpandedUrl string `json:"expanded_url"`
			Url         string `json:"url"`
			Indices     []int  `json:"indices"`
		} `json:"urls"`
		Hashtags []struct {
			Indices []int  `json:"indices"`
			Text    string `json:"text"`
		} `json:"hashtags"`
		Symbols []interface{} `json:"symbols"`
	} `json:"entities"`
	ExtendedEntities struct {
		Media []struct {
			ExtSensitiveMediaWarning struct {
				AdultContent    bool `json:"adult_content"`
				GraphicViolence bool `json:"graphic_violence"`
				Other           bool `json:"other"`
			} `json:"ext_sensitive_media_warning"`
			VideoInfo struct {
				Variants []struct {
					Bitrate int    `json:"bitrate,omitempty"`
					Url     string `json:"url"`
				} `json:"variants"`
			} `json:"video_info"`
			DisplayUrl    string `json:"display_url"`
			ExpandedUrl   string `json:"expanded_url"`
			ExtAltText    string `json:"ext_alt_text"`
			IdStr         string `json:"id_str"`
			Indices       []int  `json:"indices"`
			MediaKey      string `json:"media_key"`
			MediaUrlHttps string `json:"media_url_https"`
			Type          string `json:"type"`
			Url           string `json:"url"`
			ExtMediaColor struct {
				Palette []struct {
					Percentage float64 `json:"percentage"`
					Rgb        struct {
						Blue  int `json:"blue"`
						Green int `json:"green"`
						Red   int `json:"red"`
					} `json:"rgb"`
				} `json:"palette"`
			} `json:"ext_media_color"`
			ExtMediaAvailability struct {
				Status string `json:"status"`
			} `json:"ext_media_availability"`
			Features struct {
				Large struct {
					Faces []interface{} `json:"faces"`
				} `json:"large"`
				Medium struct {
					Faces []interface{} `json:"medium"`
				} `json:"medium"`
				Small struct {
					Faces []interface{} `json:"small"`
				} `json:"small"`
				Orig struct {
					Faces []interface{} `json:"orig"`
				} `json:"orig"`
			} `json:"features"`
			Sizes struct {
				Large struct {
					H      int    `json:"h"`
					W      int    `json:"w"`
					Resize string `json:"resize"`
				} `json:"large"`
				Medium struct {
					H      int    `json:"h"`
					W      int    `json:"w"`
					Resize string `json:"resize"`
				} `json:"medium"`
				Small struct {
					H      int    `json:"h"`
					W      int    `json:"w"`
					Resize string `json:"resize"`
				} `json:"small"`
				Thumb struct {
					H      int    `json:"h"`
					W      int    `json:"w"`
					Resize string `json:"resize"`
				} `json:"thumb"`
			} `json:"sizes"`
			OriginalInfo struct {
				Height     int `json:"height"`
				Width      int `json:"width"`
				FocusRects []struct {
					X int `json:"x"`
					Y int `json:"y"`
					H int `json:"h"`
					W int `json:"w"`
				} `json:"focus_rects"`
			} `json:"original_info"`
		} `json:"media"`
	} `json:"extended_entities"`
	FavoriteCount             int    `json:"favorite_count"`
	Favorited                 bool   `json:"favorited"`
	FullText                  string `json:"full_text"`
	IsQuoteStatus             bool   `json:"is_quote_status"`
	Lang                      string `json:"lang"`
	PossiblySensitive         bool   `json:"possibly_sensitive"`
	PossiblySensitiveEditable bool   `json:"possibly_sensitive_editable"`
	QuoteCount                int    `json:"quote_count"`
	ReplyCount                int    `json:"reply_count"`
	RetweetCount              int    `json:"retweet_count"`
	Retweeted                 bool   `json:"retweeted"`
	Source                    string `json:"source"`
	UserIdStr                 string `json:"user_id_str"`
	IdStr                     string `json:"id_str"`
}

func (tr *TweetResult) Text() string {
	text := tr.FullText

	for _, hashtag := range tr.Entities.Hashtags {
		re := regexp.MustCompile(`(#` + hashtag.Text + `)( |$|\s|[^\w])`)
		text = re.ReplaceAllString(text, `<a class="text-teal-500" href="https://twitter.com/hashtag/`+hashtag.Text+`" target="_blank" rel="noreferrer">$0</a> `)
	}
	for _, mention := range tr.Entities.UserMentions {
		re := regexp.MustCompile(`(@` + mention.ScreenName + `)( |$|\s|[^\w])`)
		text = re.ReplaceAllString(text, `<a class="text-teal-600" href="https://twitter.com/`+mention.ScreenName+`" target="_blank" rel="noreferrer">$0</a> `)
	}
	for _, _url := range tr.Entities.Urls {
		text = strings.ReplaceAll(text, _url.Url, `<a class="text-yellow-600" href="`+_url.ExpandedUrl+`" target="_blank" rel="noreferrer">`+_url.ExpandedUrl+`</a>`)
	}
	for _, _url := range tr.Entities.Media {
		text = strings.ReplaceAll(text, _url.Url, ``)
	}

	return text
}
