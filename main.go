package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/deflix-tv/go-stremio"
	"go.uber.org/zap"
)

const (
	version = "0.3.0"
)

var (
	bindAddr = flag.String("bindAddr", "localhost", `Local interface address to bind to. "localhost" only allows access from the local host. "0.0.0.0" binds to all network interfaces.`)
	port     = flag.Int("port", 8080, "Port to listen on")
	dataDir  = flag.String("dataDir", ".", "Location of the data directory. It contains CSV files with IMDb IDs and a \"metas\" subdirectory with meta JSON files")
	logLevel = flag.String("logLevel", "info", `Log level to show only logs with the given and more severe levels. Can be "debug", "info", "warn", "error"`)
	cacheAge = flag.String("cacheAge", "24h", "Max age for a client or proxy cache. The format must be acceptable by Go's 'time.ParseDuration()', for example \"24h\".")
)

var (
	manifest = stremio.Manifest{
		ID:          "tv.deflix.stremio-top-movies",
		Name:        "Top movies",
		Description: "Multiple catalogs of top movie lists: IMDb Top 250, IMDb Most Popular, Top Box Office (US), Rotten Tomatoes Certified Fresh Movies, Academy Award for Best Picture, Cannes Film Festival Palme d'Or winners, Venice Film Festival Golden Lion winners, Berlin International Film Festival Golden Bear winners",
		Version:     version,

		ResourceItems: []stremio.ResourceItem{
			{
				Name: "catalog",
			},
		},
		Types:    []string{"movie"},
		Catalogs: catalogs,

		IDprefixes: []string{"tt"},
		// Must use www.deflix.tv instead of just deflix.tv because GitHub takes care of redirecting non-www to www and this leads to HTTPS certificate issues.
		Background: "https://www.deflix.tv/images/Logo-1024px.png",
		Logo:       "https://www.deflix.tv/images/Logo-250px.png",
	}

	catalogs = []stremio.CatalogItem{
		{
			Type: "movie",
			ID:   "imdb-top-250",
			Name: "IMDb Top Rated (a.k.a. Top 250)",
		},
		{
			Type: "movie",
			ID:   "imdb-most-popular",
			Name: "IMDb Most Popular",
		},
		{
			Type: "movie",
			ID:   "top-box-office-us",
			Name: "Top Box Office (US, last weekend)",
		},
		{
			Type: "movie",
			ID:   "rt-certified-fresh",
			Name: "Rotten Tomatoes Certified Fresh (DVD & Streaming)",
		},
		{
			Type: "movie",
			ID:   "academy-awards-winners",
			Name: "Academy Award for Best Picture winners",
		},
		{
			Type: "movie",
			ID:   "palme-dor-winners",
			Name: "Cannes Film Festival Palme d'Or winners",
		},
		{
			Type: "movie",
			ID:   "golden-lion-winners",
			Name: "Venice Film Festival Golden Lion winners",
		},
		{
			Type: "movie",
			ID:   "golden-bear-winners",
			Name: "Berlin International Film Festival Golden Bear winners",
		},
	}
)

const (
	redirectURL = "https://www.deflix.tv"
)

var (
	responses = make(map[string][]stremio.MetaPreviewItem, len(catalogs))
)

func init() {
	// Timeout for global default HTTP client (for when using `http.Get()`)
	http.DefaultClient.Timeout = 5 * time.Second
}

func main() {
	flag.Parse()

	// Prep

	logger, err := stremio.NewLogger(*logLevel)
	if err != nil {
		panic(err)
	}

	cacheAgeDuration, err := time.ParseDuration(*cacheAge)
	if err != nil {
		logger.Fatal("Couldn't parse cacheAge", zap.Error(err))
	}
	logger.Info("Cache age set", zap.Duration("duration", cacheAgeDuration))
	// Clean input
	if strings.HasSuffix(*dataDir, "/") {
		*dataDir = strings.TrimRight(*dataDir, "/")
	}

	// Initialize catalogs

	logger.Info("Initializing catalogs...")
	for _, catalogItem := range catalogs {
		id := catalogItem.ID
		responses[id] = createCatalogResponse(id, logger)
	}
	logger.Info("Initialized catalogs")

	// Set up addon

	catalogHandlers := map[string]stremio.CatalogHandler{"movie": movieHandler}
	options := stremio.Options{
		BindAddr:            *bindAddr,
		Port:                *port,
		Logger:              logger,
		RedirectURL:         redirectURL,
		CacheAgeCatalogs:    cacheAgeDuration,
		CachePublicCatalogs: true,
		HandleEtagCatalogs:  true,
	}
	addon, err := stremio.NewAddon(manifest, catalogHandlers, nil, options)
	if err != nil {
		logger.Fatal("Couldn't create addon", zap.Error(err))
	}

	// Go!

	addon.Run()
}

func createCatalogResponse(catalog string, logger *zap.Logger) []stremio.MetaPreviewItem {
	var result []stremio.MetaPreviewItem

	records := readCSV(*dataDir+"/"+catalog+".csv", logger)
	metas := readMetas(records, *dataDir+"/metas", logger)
	for _, meta := range metas {
		var item stremio.MetaPreviewItem
		if err := json.Unmarshal(meta, &item); err != nil {
			logger.Warn("Couldn't unmarshal meta JSON into stremio.MetaPreviewItem", zap.Error(err))
		}
		result = append(result, item)
	}

	return result
}

func readCSV(filePath string, logger *zap.Logger) [][]string {
	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		logger.Fatal("Couldn't read file", zap.Error(err))
	}
	csvReader := csv.NewReader(bytes.NewReader(fileBytes))
	records, err := csvReader.ReadAll()
	if err != nil {
		logger.Fatal("Couldn't read CSV", zap.Error(err))
	}
	return records
}

func readMetas(records [][]string, metasDir string, logger *zap.Logger) [][]byte {
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
		logger.Fatal("Couldn't find \"IMDb ID\" in CSV header", zap.Strings("csvHeader", headRecord))
	}

	var result [][]byte
	for _, record := range records[1:] {
		imdbID := record[imdbIndex]
		// We assume that the metafetcher has been used to already write all meta JSON files for all required IMDb IDs to the directory, so we can directly read the files here via the IMDb ID + ".json", instead of going through the actual files and only read it when it matches one of our IMDb IDs.
		fileContent, err := ioutil.ReadFile(metasDir + "/" + imdbID + ".json")
		if err != nil {
			logger.Warn("Couldn't read meta file for IMDb ID", zap.String("imdbID", imdbID), zap.Error(err))
			continue
		}
		result = append(result, fileContent)
	}

	return result
}
