package main

import (
	"flag"
	"strings"
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

	imdbClient := newIMDbClient()
	rtClient := newRTclient(imdbClient)
	imdbClient.scrapeTop250(*dataDir + "/imdb-top-250.csv")
	imdbClient.scrapeMostPopular(*dataDir + "/imdb-most-popular.csv")
	imdbClient.scrapeBoxOfficeUSWeekend(*dataDir + "/top-box-office-us.csv")
	rtClient.scrapeCertifiedFreshDVDstreaming(*dataDir + "/rt-certified-fresh.csv")
}
