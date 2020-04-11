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

	scrapeIMDbTop250(&httpClient, *dataDir+"/imdb-top-250.csv")
	scrapeIMDbMostPopular(&httpClient, *dataDir+"/imdb-most-popular.csv")
	scrapeBoxOfficeWeekendUS(&httpClient, *dataDir+"/top-box-office-us.csv")
	scrapeRTcertifiedFreshDVDstreaming(&httpClient, *dataDir+"/rt-certified-fresh.csv")
	scrapeWikipediaAcademyAwardWinners(&httpClient, *dataDir+"/academy-awards-winners.csv")
	scrapeWikipediaPalmeDorWinners(&httpClient, *dataDir+"/palme-dor-winners.csv")
}
