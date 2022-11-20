package scraper

type ConversationResponse struct {
	GlobalObjects struct {
		Tweets map[string]TweetResult      `json:"tweets"`
		Users  map[string]ConversationUser `json:"users"`
	} `json:"globalObjects"`
	Timeline struct {
		Instructions []struct {
			AddEntries struct {
				Entries []struct {
					Content struct {
						Item struct {
							Content struct {
								Tweet struct {
									ID string `json:"id"`
								} `json:"tweet"`
								User struct {
									ID string `json:"id"`
								} `json:"user"`
							} `json:"content"`
						} `json:"item"`
						Operation struct {
							Cursor struct {
								Value      string `json:"value"`
								CursorType string `json:"cursorType"`
							} `json:"cursor"`
						} `json:"operation"`
						TimelineModule struct {
							Items []struct {
								Item struct {
									ClientEventInfo struct {
										Details struct {
											GuideDetails struct {
												TransparentGuideDetails struct {
													TrendMetadata struct {
														TrendName string `json:"trendName"`
													} `json:"trendMetadata"`
												} `json:"transparentGuideDetails"`
											} `json:"guideDetails"`
										} `json:"details"`
									} `json:"clientEventInfo"`
								} `json:"item"`
							} `json:"items"`
						} `json:"timelineModule"`
					} `json:"content,omitempty"`
				} `json:"entries"`
			} `json:"addEntries"`
			PinEntry struct {
				Entry struct {
					Content struct {
						Item struct {
							Content struct {
								Tweet struct {
									ID string `json:"id"`
								} `json:"tweet"`
							} `json:"content"`
						} `json:"item"`
					} `json:"content"`
				} `json:"entry"`
			} `json:"pinEntry,omitempty"`
			ReplaceEntry struct {
				Entry struct {
					Content struct {
						Operation struct {
							Cursor struct {
								Value      string `json:"value"`
								CursorType string `json:"cursorType"`
							} `json:"cursor"`
						} `json:"operation"`
					} `json:"content"`
				} `json:"entry"`
			} `json:"replaceEntry,omitempty"`
		} `json:"instructions"`
	} `json:"timeline"`
}

type ConversationUser struct {
	CreatedAt   string `json:"created_at"`
	Description string `json:"description"`
	Entities    struct {
		Url struct {
			Urls []struct {
				ExpandedUrl string `json:"expanded_url"`
			} `json:"urls"`
		} `json:"url"`
	} `json:"entities"`
	FavouritesCount      int      `json:"favourites_count"`
	FollowersCount       int      `json:"followers_count"`
	FriendsCount         int      `json:"friends_count"`
	IdStr                string   `json:"id_str"`
	ListedCount          int      `json:"listed_count"`
	Name                 string   `json:"name"`
	Location             string   `json:"location"`
	PinnedTweetIdsStr    []string `json:"pinned_tweet_ids_str"`
	ProfileBannerUrl     string   `json:"profile_banner_url"`
	ProfileImageUrlHttps string   `json:"profile_image_url_https"`
	Protected            bool     `json:"protected"`
	ScreenName           string   `json:"screen_name"`
	StatusesCount        int      `json:"statuses_count"`
	Verified             bool     `json:"verified"`
}
type Place struct {
	ID          string `json:"id"`
	PlaceType   string `json:"place_type"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	CountryCode string `json:"country_code"`
	Country     string `json:"country"`
	BoundingBox struct {
		Type        string        `json:"type"`
		Coordinates [][][]float64 `json:"coordinates"`
	} `json:"bounding_box"`
}

func (c *ConversationResponse) GetUser(userId string) *ConversationUser {
	for key, user := range c.GlobalObjects.Users {
		if key == userId || user.IdStr == userId {
			return &user
		}
	}
	return &ConversationUser{}
}
