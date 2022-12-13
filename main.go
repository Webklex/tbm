package main

import (
	"embed"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"tbm/app"
	"tbm/utils/log"
	"time"
)

//go:embed static
var staticFiles embed.FS

var buildNumber = "custom"
var buildVersion = "0.0.0"

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	a := app.NewApplication(staticFiles)

	flag.StringVar(&a.ConfigFileName, "config", a.ConfigFileName, "Application config file")
	flag.StringVar(&a.DataDir, "data-dir", a.DataDir, "Folder containing all fetched data")

	flag.StringVar(&a.Server.Host, "host", a.Server.Host, "Host address the api should bind to")
	flag.UintVar(&a.Server.Port, "port", a.Server.Port, "Port the api should bind to")

	flag.StringVar(&a.Scraper.Cookie, "cookie", a.Scraper.Cookie, "Twitter cookie string")

	flag.StringVar(&a.Scraper.AccessToken, "access-token", a.Scraper.AccessToken, "Twitter bearer access token")
	flag.StringVar(&a.Scraper.Sections.Index, "index-section", a.Scraper.Sections.Index, "Twitter bookmark api section name")
	flag.StringVar(&a.Scraper.Sections.Remove, "remove-section", a.Scraper.Sections.Remove, "Twitter remove bookmark api section name")
	flag.DurationVar(&a.Scraper.Timeout, "timeout", a.Scraper.Timeout, "Request timeout")
	flag.DurationVar(&a.Scraper.Delay, "delay", a.Scraper.Delay, "Delay your request by a given time")

	flag.BoolVar(&a.Danger.RemoveBookmarks, "danger-remove-bookmarks", a.Danger.RemoveBookmarks, "Remove the bookmark on Twitter if the tweet has been downloaded")

	flag.IntVar(&log.Mode, "log", log.Mode, "Set the log mode (0 = all, 1 = success, 2 = warning, 3 = statistic, 4 = error)")

	sv := flag.Bool("version", false, "Show version and exit")
	nc := flag.Bool("no-color", false, "Disable color output")
	offline := flag.Bool("offline", false, "Don't fetch new bookmarks; link to local files only")
	flag.Parse()

	if *nc {
		color.NoColor = true // disables colorized output
	}

	a.Build = app.Build{
		Number:  buildNumber,
		Version: buildVersion,
	}

	if *sv {
		fmt.Printf("version: %s\nbuild number: %s\n", color.CyanString(buildVersion), color.CyanString(buildNumber))
		os.Exit(0)
	}
	if *offline {
		a.Mode = app.OfflineMode
	}

	if err := a.Load(); err != nil {
		log.Error("Failed to load the config file: %s", err.Error())
		os.Exit(2) // No such file or directory
	}

	if err := a.Start(); err != nil {
		log.Error("Failed to start the application: %s", err.Error())
		os.Exit(131) // State not recoverable
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	if err := a.Stop(); err != nil {
		log.Error("Failed to shutdown: %s", err.Error())
		os.Exit(131) // State not recoverable
	}
	os.Exit(1)
}
