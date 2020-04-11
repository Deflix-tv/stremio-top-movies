package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

var (
	dataDir = flag.String("dataDir", ".", "Location of the data directory. It contains CSV files with IMDb IDs and a \"metas\" subdirectory will be used for writing metas as JSOn files.")
)

var (
	lists = []string{
		"imdb-top-250",
		"imdb-most-popular",
		"top-box-office-us",
		"rt-certified-fresh",
		"academy-awards-winners",
		"palme-dor-winners",
		"golden-lion-winners",
		"golden-bear-winners",
	}
)

func init() {
	// Timeout for global default HTTP client (for when using `http.Get()`)
	http.DefaultClient.Timeout = 5 * time.Second
}

func main() {
	flag.Parse()

	// Clean input
	if strings.HasSuffix(*dataDir, "/") {
		*dataDir = strings.TrimRight(*dataDir, "/")
	}

	for _, list := range lists {
		records := read(*dataDir + "/" + list + ".csv")
		missingMetas := determineMissingMetas(records, *dataDir+"/metas")
		metas := fetchMetas(missingMetas)
		writeMetas(metas, *dataDir+"/metas")
	}
}

func read(filePath string) [][]string {
	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal("Couldn't read file:", err)
	}
	csvReader := csv.NewReader(bytes.NewReader(fileBytes))
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal("Couldn't read CSV:", err)
	}
	return records
}

func determineMissingMetas(records [][]string, metasDir string) []string {
	headRecord := records[0]
	imdbIndex := 0
	found := false
	for ; imdbIndex < len(headRecord); imdbIndex++ {
		if headRecord[imdbIndex] == "IMDb ID" {
			found = true
			break
		}
	}
	if !found {
		log.Fatal("Couldn't find \"IMDb ID\" in CSV header:", headRecord)
	}

	var imdbIDs []string
	for _, record := range records[1:] {
		imdbIDs = append(imdbIDs, record[imdbIndex])
	}

	// Now we have *all* IMDb IDs. But we only want to know the missing ones.
	fileInfos, err := ioutil.ReadDir(metasDir)
	if err != nil {
		log.Fatal("Couldn't read metas directory:", err)
	}
	var fileNames []string
	for _, fileInfo := range fileInfos {
		fileName := strings.TrimSuffix(fileInfo.Name(), ".json")
		fileNames = append(fileNames, fileName)
	}
	var result []string
	for _, imdbID := range imdbIDs {
		found = false
		for _, fileName := range fileNames {
			if fileName == imdbID {
				found = true
				break
			}
		}
		if !found {
			result = append(result, imdbID)
		}
	}

	return result
}

type meta struct {
	imdbID string
	meta   string
}

func fetchMetas(imdbIDs []string) []meta {
	var result []meta
	for _, imdbID := range imdbIDs {
		log.Println("Fetching meta for", imdbID)
		// Note: Add "?sda" to invalidate the server's cache
		url := "https://v3-cinemeta.strem.io/meta/movie/" + imdbID + ".json"
		res, err := http.Get(url)
		if err != nil {
			log.Printf("Couldn't GET %v: %v", url, err)
			continue
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			log.Println("Bad GET response:", res.StatusCode)
			continue
		}
		resBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Println("Couldn't read response body:", err)
			continue
		}
		metaJSON := gjson.GetBytes(resBody, "meta").Raw
		if metaJSON == "" {
			log.Println("Response body is empty or at least doesn't contain a \"meta\" element")
			continue
		}
		result = append(result, meta{imdbID, metaJSON})
		// Don't DoS the server
		time.Sleep(100 * time.Millisecond)
	}
	return result
}

func writeMetas(metas []meta, metaDir string) {
	for _, meta := range metas {
		log.Println("Write meta file for", meta.imdbID)
		if err := ioutil.WriteFile(metaDir+"/"+meta.imdbID+".json", []byte(meta.meta), 0600); err != nil {
			log.Fatal("Couldn't write file:", err)
		}
	}
}
