package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type WikiClient struct {
	httpClient *http.Client
}

func newWikiClient() WikiClient {
	return WikiClient{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c WikiClient) scrapeAcademyAwardWinners(filePath string) {
	req, _ := http.NewRequest("GET", "https://en.wikipedia.org/wiki/List_of_Academy_Award-winning_films", nil)
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

	record := []string{"year", "title", "IMDb ID"}
	if err = csvWriter.Write(record); err != nil {
		log.Fatal(err)
	}

	var records [][]string

	// Find the elements of the list and write them into the CSV
	doc.Find(".wikitable tbody tr").Each(func(i int, s *goquery.Selection) {
		// Only the main Academy Award winners have a yellow background
		if val, ok := s.Attr("style"); !ok || val == "" {
			return
		}
		title := s.Find("a").First().Text()
		href, _ := s.Find("a").First().Attr("href")
		url := "https://en.wikipedia.org" + href
		year := s.Find("a").Eq(1).Text()

		req, _ = http.NewRequest("GET", url, nil)
		res, err := c.httpClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
		}

		// Load the HTML document
		doc, err = goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			log.Fatal(err)
		}

		var imdbID string
		doc.Find("#External_links").Parent().Siblings().Find("ul li a").Each(func(i int, s *goquery.Selection) {
			href, ok := s.Attr("href")
			if !ok || (!strings.HasPrefix(href, "https://www.imdb.com/title/") && !strings.HasPrefix(href, "http://www.imdb.com/title/")) || !strings.Contains(s.Text(), title) {
				return
			}
			splitElems := strings.Split(href, "/")
			// It's probably not always `splitElems[len(splitElems)-1]`, because some might put the link without trailing `/`, some with.
			for _, splitElem := range splitElems {
				if strings.HasPrefix(splitElem, "tt") {
					imdbID = splitElem
					break
				}
			}
			if imdbID == "" {
				log.Fatal("Couldn't determine IMDb ID from the movies Wikipedia page:", title)
			}
		})

		record := []string{year, title, imdbID}
		records = append(records, record)
	})

	// Sort records (they're sorted up to the half of the list on Wikipedia, then unsorted)
	for year := time.Now().Year(); year >= 1927; year-- {
		for _, record := range records {
			if strconv.Itoa(year) == record[0] {
				fmt.Printf("%v: %v; ID: %v\n", record[0], record[1], record[2])

				if err = csvWriter.Write(record); err != nil {
					log.Fatal(err)
				}
				break
			}

		}
	}
}
