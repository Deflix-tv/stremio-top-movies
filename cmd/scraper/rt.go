package main

import (
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/tidwall/gjson"
)

var rtFreshRegex = regexp.MustCompile(`\[\{"id":.*`)

type getIDer interface {
	getID(title string) string
}

type RTclient struct {
	httpClient *http.Client
	idGetter   getIDer
}

func newRTclient(idGetter getIDer) RTclient {
	return RTclient{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		idGetter: idGetter,
	}
}

func (c RTclient) scrapeCertifiedFreshDVDstreaming(filePath string) {
	req, _ := http.NewRequest("GET", "https://www.rottentomatoes.com/browse/cf-dvd-streaming-all", nil)
	// Set language just to make sure it's not like with IMDb where the movie names are depending on the country of the request's IP.
	req.Header.Add("accept-language", "en-US")
	res, err := c.httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
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

	// Find the elements of the list and write them into the CSV.
	// The delivered HTML is not what is shown in a browser, because some content is only added afterwards via JavaScript.
	// So we take the content from the JavaScript.
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal("Couldn't read body:", err)
	}
	freshJSON := rtFreshRegex.Find(body)
	// Cut away the trailing `,`
	freshJSON = freshJSON[:len(freshJSON)-1]

	titles := gjson.GetBytes(freshJSON, "#.title").Array()
	rank := 0
	for _, titleResult := range titles {
		rank += 1
		title := titleResult.String()
		// Search for title on IMDb to get its IMDb ID
		imdbID := c.idGetter.getID(title)
		if imdbID == "" {
			log.Fatal("Couldn't determine IMDb ID for title:", title)
		}

		fmt.Printf("%v. %v; ID: %v\n", rank, title, imdbID)

		record := []string{strconv.Itoa(rank), title, imdbID}
		if err = csvWriter.Write(record); err != nil {
			log.Fatal(err)
		}
	}
}
