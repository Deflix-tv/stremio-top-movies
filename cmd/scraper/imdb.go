package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/tidwall/gjson"
)

type IMDbClient struct {
	httpClient *http.Client
}

func newIMDbClient() IMDbClient {
	return IMDbClient{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c IMDbClient) scrapeTop250(filePath string) {
	req, _ := http.NewRequest("GET", "https://www.imdb.com/chart/top/", nil)
	// Must set language, otherwise IMDb determines the language based on IP and then movie names are language-specific.
	req.Header.Add("accept-language", "en-US")
	res, err := c.httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	csvWriter := csv.NewWriter(f)
	defer csvWriter.Flush()

	record := []string{"rank", "title", "year", "IMDb ID"}
	if err = csvWriter.Write(record); err != nil {
		log.Fatal(err)
	}

	// Find the elements of the list and write them into the CSV
	doc.Find(".lister-list tr").Each(func(i int, s *goquery.Selection) {
		rank := i + 1
		title := s.Find(".titleColumn a").Text()
		href, _ := s.Find(".titleColumn a").Attr("href")
		year := s.Find(".titleColumn span").Text()
		year = strings.Trim(year, "()")
		id := strings.Split(href, "/")[2]

		fmt.Printf("%v. %v (%v); ID: %v\n", rank, title, year, id)

		record := []string{strconv.Itoa(rank), title, year, id}
		if err = csvWriter.Write(record); err != nil {
			log.Fatal(err)
		}
	})
}

func (c IMDbClient) scrapeMostPopular(filePath string) {
	req, _ := http.NewRequest("GET", "https://www.imdb.com/chart/moviemeter", nil)
	// Must set language, otherwise IMDb determines the language based on IP and then movie names are language-specific.
	req.Header.Add("accept-language", "en-US")
	res, err := c.httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	csvWriter := csv.NewWriter(f)
	defer csvWriter.Flush()

	record := []string{"rank", "title", "year", "IMDb ID"}
	if err = csvWriter.Write(record); err != nil {
		log.Fatal(err)
	}

	// Find the elements of the list and write them into the CSV
	doc.Find(".lister-list tr").Each(func(i int, s *goquery.Selection) {
		rank := i + 1
		title := s.Find(".titleColumn a").Text()
		href, _ := s.Find(".titleColumn a").Attr("href")
		year := s.Find(".titleColumn .secondaryInfo").Text()
		// Although the HTML doesn't look like this, goquery returns something like this: `(2020)(\n\n4)`
		year = strings.TrimLeft(year, "(")
		year = year[:4]
		id := strings.Split(href, "/")[2]

		fmt.Printf("%v. %v (%v); ID: %v\n", rank, title, year, id)

		record := []string{strconv.Itoa(rank), title, year, id}
		if err = csvWriter.Write(record); err != nil {
			log.Fatal(err)
		}
	})
}

func (c IMDbClient) scrapeBoxOfficeUSWeekend(filePath string) {
	req, _ := http.NewRequest("GET", "https://www.imdb.com/chart/boxoffice", nil)
	// Must set language, otherwise IMDb determines the language based on IP and then movie names are language-specific.
	req.Header.Add("accept-language", "en-US")
	res, err := c.httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	csvWriter := csv.NewWriter(f)
	defer csvWriter.Flush()

	record := []string{"rank", "title", "IMDb ID"}
	if err = csvWriter.Write(record); err != nil {
		log.Fatal(err)
	}

	// Find the elements of the list and write them into the CSV
	doc.Find(".chart tbody tr").Each(func(i int, s *goquery.Selection) {
		rank := i + 1
		title := s.Find(".titleColumn a").Text()
		href, _ := s.Find(".titleColumn a").Attr("href")
		id := strings.Split(href, "/")[2]
		id = strings.Split(id, "?")[0]

		fmt.Printf("%v. %v; ID: %v\n", rank, title, id)

		record := []string{strconv.Itoa(rank), title, id}
		if err = csvWriter.Write(record); err != nil {
			log.Fatal(err)
		}
	})
}

func (c IMDbClient) scrapePalmeDorWinners(filePath string) {
	c.scrapeEvent("ev0000147", 1939, filePath)
}

func (c IMDbClient) scrapeGoldenLionWinners(filePath string) {
	c.scrapeEvent("ev0000681", 1946, filePath)
}

func (c IMDbClient) scrapeGoldenBearWinners(filePath string) {
	c.scrapeEvent("ev0000091", 1951, filePath)
}

func (c IMDbClient) scrapeEvent(eventID string, startYear int, filePath string) {
	f, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	csvWriter := csv.NewWriter(f)
	defer csvWriter.Flush()

	record := []string{"year", "title", "IMDb ID"}
	if err = csvWriter.Write(record); err != nil {
		log.Fatal(err)
	}

	for year := time.Now().Year(); year >= startYear; year-- {
		yearString := strconv.Itoa(year)

		req, _ := http.NewRequest("GET", "https://www.imdb.com/event/"+eventID+"/"+yearString+"/1/", nil)
		// Must set language, otherwise IMDb determines the language based on IP and then movie names are language-specific.
		// TODO: Doesn't seem to help here!
		req.Header.Add("accept-language", "en-US")
		res, err := c.httpClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			// Some years in the past didn't have winners
			//log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
			continue
		}

		reader := bufio.NewReader(res.Body)
		var json string
		for line, err := reader.ReadString('\n'); err == nil; line, err = reader.ReadString('\n') {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "IMDbReactWidgets.NomineesWidget.push") {
				line = strings.TrimPrefix(line, "IMDbReactWidgets.NomineesWidget.push(['center-3-react',")
				line = strings.TrimSpace(line)
				json = strings.TrimSuffix(line, "]);")
			}
		}

		title := gjson.Get(json, "nomineesWidgetModel.eventEditionSummary.awards.0.categories.0.nominations.0.primaryNominees.0.name").String()
		imdbID := gjson.Get(json, "nomineesWidgetModel.eventEditionSummary.awards.0.categories.0.nominations.0.primaryNominees.0.const").String()

		fmt.Printf("%v: %v; ID: %v\n", yearString, title, imdbID)

		record := []string{yearString, title, imdbID}
		if err = csvWriter.Write(record); err != nil {
			log.Fatal(err)
		}
	}
}

func (c IMDbClient) getID(title string) string {
	title = url.QueryEscape(title)
	req, _ := http.NewRequest("GET", "https://www.imdb.com/find?q="+title, nil)
	// Must set language, otherwise IMDb determines the language based on IP and then movie names are language-specific.
	req.Header.Add("accept-language", "en-US")
	res, err := c.httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	var id string
	// Find the elements of the list and write them into the CSV
	doc.Find(".result_text").Each(func(i int, s *goquery.Selection) {
		// We only care about the first result
		if i > 0 {
			return
		}
		href, _ := s.Find("a").Attr("href")
		id = strings.Split(href, "/")[2]
	})
	return id
}
