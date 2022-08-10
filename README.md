# TBM - Twitter Bookmark Manager

Fetch all your bookmarked tweets and make them accessible through a webinterface.

![Search Tweets](.github/images/search_tweets.png)
(Search for bookmarked tweets)

## Usage
In order to fetch your bookmarks, you'll have to supply an active access token with a matching cookie. 
You can get both by the following steps:
1. Login to twitter.com and go to https://twitter.com/i/bookmarks
2. Press `f12`, switch to the `Network` tab and look for a request named `Bookmarks?variables=%7B%22count%22%3A20..`
3. Switch to the `Headers` tab if it isn't selected and scroll down to `Request Headers`
4. Copy the line starting with `cookie: ` and `authorization: Bearer `
5. Check if the `section` has changed (part of the url in front of `Bookmarks?variables=%7B%22count%22%3A20..`), if so copy it as well

```bash
Usage of tbm:
  -access-token string
        Twitter bearer access token
  -config string
        Application config file (default "./config/config.json")
  -cookie string
        Twitter cookie string
  -data-dir string
        Folder containing all fetched data (default "./data")
  -host string
        Host address the api should bind to (default "localhost")
  -port uint
        Port the api should bind to (default 4788)
  -section string
        Twitter bookmark api section name (default "BvX-1Exs_MDBeKAedv2T_w")
  -timezone string
        Application time zone (default "UTC")
  -version
        Show version and exit
```

## Configuration
Besides the command arguments, you can also provide a config file:
```json
{
  "timezone": "UTC",
  "data_dir": "./data",
  "server": {
    "host": "localhost",
    "port": 4788
  },
  "scraper": {
    "section": "BvX-1Exs_MDBeKAedv2T_w",
    "cookie": "guest_id=...",
    "access_token": "AAAAA..."
  }
}
```

## Websocket commands
The websocket can be accessed under `ws://{host}:{port}/ws`.

Set the access token and cookie:
```json
{
  "command":"set_tokens",
  "payload":{
    "cookie": "guest_id=...",
    "access_token": "AAAAA..."
  }
}
```
Get all tweets:
```json
{
  "command":"get_tweets",
  "payload":{}
}
```
Search for tweets containing the search query:
```json
{
  "command":"search_tweets",
  "payload":{
    "query": "foo bar"
  }
}
```

## Custom Styles
By default all assets (.js, .css, .html, etc) get included while building a new version.

### Structure:
- gui
  - css
    - tailwind.css
- public
  - css
    - tailwind.css (compiled tailwind css)
    - style.css (custom styling)
  - js
    - app.js
  - index.html

## Development
Requirements:
- `Node` v12.13
- `Golang` ^1.17.2

```bash
npm run watch
npm run build
go run main.go
```

## Build
Build a new release:
```bash
./build.sh
```

Build a new regular binary:
```bash
go build -ldflags "-w -s -X main.buildNumber=1 -X main.buildVersion=custom" -o tbm
```
