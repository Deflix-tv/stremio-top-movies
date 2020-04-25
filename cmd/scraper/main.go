package main

import (
	"flag"
	"strings"
)

var (
	dataDir   = flag.String("dataDir", ".", "Location of the data directory. This is where the CSV files will be written.")
	festivals = flag.Bool("festivals", false, "Include scraping film festival data.")
)

func main() {
	flag.Parse()

	// Clean input
	if strings.HasSuffix(*dataDir, "/") {
		*dataDir = strings.TrimRight(*dataDir, "/")
	}

	imdbClient := newIMDbClient()
	rtClient := newRTclient(imdbClient)
	wikiClient := newWikiClient()
	imdbClient.scrapeTop250(*dataDir + "/imdb-top-250.csv")
	imdbClient.scrapeMostPopular(*dataDir + "/imdb-most-popular.csv")
	imdbClient.scrapeBoxOfficeUSWeekend(*dataDir + "/top-box-office-us.csv")
	rtClient.scrapeCertifiedFreshDVDstreaming(*dataDir + "/rt-certified-fresh.csv")
	if *festivals {
		wikiClient.scrapeAcademyAwardWinners(*dataDir + "/academy-awards-winners.csv")
		imdbClient.scrapePalmeDorWinners(*dataDir + "/palme-dor-winners.csv")
		imdbClient.scrapeGoldenLionWinners(*dataDir + "/golden-lion-winners.csv")
		imdbClient.scrapeGoldenBearWinners(*dataDir + "/golden-bear-winners.csv")
	}
}
