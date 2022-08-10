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

//go:embed public
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
	flag.StringVar(&a.Scraper.Section, "section", a.Scraper.Section, "Twitter bookmark api section name")

	sv := flag.Bool("version", false, "Show version and exit")
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

	if err := a.Load(); err != nil {
		fmt.Printf("Failed to load the config file: %s\n", err.Error())
		os.Exit(2) // No such file or directory
	}

	if err := a.Start(); err != nil {
		fmt.Printf("Failed to start the application: %s\n", err.Error())
		os.Exit(131) // State not recoverable
	}

}