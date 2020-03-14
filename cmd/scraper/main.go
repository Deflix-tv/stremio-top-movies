package main

import (
	"flag"
	"net/http"
	"strings"
	"time"
)

var (
	dataDir = flag.String("dataDir", ".", "Location of the data directory. This is where the CSV files will be written.")
)

func main() {
	flag.Parse()

	// Clean input
	if strings.HasSuffix(*dataDir, "/") {
		*dataDir = strings.TrimRight(*dataDir, "/")
	}

	httpClient := http.Client{
		Timeout: 5 * time.Second,
	}

	scrapeIMDbTop250(&httpClient)
	scrapeIMDbMostPopular(&httpClient)
	scrapeBoxOfficeWeekendUS(&httpClient)
	scrapeRTcertifiedFreshDVDstreaming(&httpClient)
	scrapeWikipediaAcademyAwardWinners(&httpClient)
	scrapeWikipediaPalmeDorWinners(&httpClient)
}
