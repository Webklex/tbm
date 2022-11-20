package main

import (
	"embed"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"tbm/app"
	"time"
)

//go:embed static
var staticFiles embed.FS

var buildNumber string
var buildVersion string

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	a := app.NewApplication(staticFiles)

	flag.StringVar(&a.Timezone, "timezone", a.Timezone, "Application time zone")
	flag.StringVar(&a.ConfigFileName, "config", a.ConfigFileName, "Application config file")
	flag.StringVar(&a.DataDir, "data-dir", a.DataDir, "Folder containing all fetched data")
	flag.StringVar(&a.Server.Host, "host", a.Server.Host, "Host address the api should bind to")
	flag.UintVar(&a.Server.Port, "port", a.Server.Port, "Port the api should bind to")
	flag.StringVar(&a.Scraper.AccessToken, "access-token", a.Scraper.AccessToken, "Twitter bearer access token")
	flag.StringVar(&a.Scraper.Cookie, "cookie", a.Scraper.Cookie, "Twitter cookie string")
	flag.StringVar(&a.Scraper.Sections.Index, "index-section", a.Scraper.Sections.Index, "Twitter bookmark api section name")
	flag.StringVar(&a.Scraper.Sections.Remove, "remove-section", a.Scraper.Sections.Remove, "Twitter remove bookmark api section name")
	flag.DurationVar(&a.Scraper.Timeout, "timeout", a.Scraper.Timeout, "Request timeout")
	flag.DurationVar(&a.Scraper.Delay, "delay", a.Scraper.Delay, "Delay your request by a given time")
	flag.BoolVar(&a.Danger.RemoveBookmarks, "danger-remove-bookmarks", a.Danger.RemoveBookmarks, "Remove the bookmark on Twitter if the tweet has been downloaded")

	sv := flag.Bool("version", false, "Show version and exit")
	offline := flag.Bool("offline", false, "Don't fetch new bookmarks; link to local files only")
	flag.Parse()

	a.Build = app.Build{
		Number:  buildNumber,
		Version: buildVersion,
	}

	if *sv {
		fmt.Printf("Version: %s\n", a.Build.Version)
		fmt.Printf("Build number: %s\n", a.Build.Number)
		return
	}
	if *offline {
		a.Mode = app.OfflineMode
	}

	if err := a.Load(); err != nil {
		fmt.Printf("Failed to load the config file: %s\n", err.Error())
		os.Exit(2) // No such file or directory
	}

	if err := a.Start(); err != nil {
		fmt.Printf("Failed to start the application: %s\n", err.Error())
		os.Exit(131) // State not recoverable
	}

}
