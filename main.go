package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/doingodswork/deflix-stremio/pkg/stremio"
)

const (
	version = "0.1.0"
)

var (
	dataDir = flag.String("dataDir", ".", "Location of the data directory. It contains CSV files with IMDb IDs and a \"metas\" subdirectory with meta JSON files")
)

var (
	manifest = stremio.Manifest{
		ID:          "tv.deflix.stremio-top-movies",
		Name:        "Top movies",
		Description: "Multiple catalogs of top movie lists: IMDb Top 250, IMDb Most Popular, Top Box Office (US), Rotten Tomatoes Certified Fresh Movies, Academy Award for Best Picture, Cannes Film Festival Palme d'Or winners",
		Version:     version,

		ResourceItems: resources,
		Types:         []string{"movie"},
		Catalogs:      catalogs,

		IDprefixes: []string{"tt"},
		// Must use www.deflix.tv instead of just deflix.tv because GitHub takes care of redirecting non-www to www and this leads to HTTPS certificate issues.
		Background: "https://www.deflix.tv/images/Logo-1024px.png",
		Logo:       "https://www.deflix.tv/images/Logo-250px.png",
	}

	resources = []stremio.ResourceItem{
		stremio.ResourceItem{
			Name: "catalog",
		},
	}

	catalogs = []stremio.CatalogItem{
		stremio.CatalogItem{
			Type: "movie",
			ID:   "imdb-top-250",
			Name: "IMDb Top Rated (a.k.a. \"IMDb Top 250\")"},
		stremio.CatalogItem{
			Type: "movie",
			ID:   "imdb-most-popular",
			Name: "IMDb Most Popular"},
		stremio.CatalogItem{
			Type: "movie",
			ID:   "top-box-office-us",
			Name: "Top Box Office (US) (last weekend)"},
		stremio.CatalogItem{
			Type: "movie",
			ID:   "rt-certified-fresh",
			Name: "Rotten Tomatoes Certified Fresh (DVD + Streaming)"},
		stremio.CatalogItem{
			Type: "movie",
			ID:   "academy-awards-winners",
			Name: "Academy Award for Best Picture"},
		stremio.CatalogItem{
			Type: "movie",
			ID:   "palme-dor-winners",
			Name: "Cannes Film Festival Palme d'Or winners"},
	}
)

const (
	addr        = "localhost:8080"
	redirectURL = "https://www.deflix.tv"
)

var (
	imdbTop250CatalogResponse       []byte
	imdbMostPopularCatalogResponse  []byte
	boxOfficeUScatalogResponse      []byte
	rtCertifiedFreshCatalogResponse []byte
	academyAwardsCatalogResponse    []byte
	palmeDorCatalogResponse         []byte
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

	log.Println("Initializing catalogs...")
	imdbTop250CatalogResponse = createCatalogResponse("imdb-top-250")
	imdbMostPopularCatalogResponse = createCatalogResponse("imdb-most-popular")
	boxOfficeUScatalogResponse = createCatalogResponse("box-office-weekend-us")
	rtCertifiedFreshCatalogResponse = createCatalogResponse("rt-certified-fresh-dvd-streaming")
	academyAwardsCatalogResponse = createCatalogResponse("academy-awards-winners")
	palmeDorCatalogResponse = createCatalogResponse("palme-dor-winners")
	log.Println("Initialized catalogs")

	log.Println("Setting up server...")
	r := mux.NewRouter()
	s := r.Methods("GET").Subrouter()
	s.Use(timerMiddleware,
		corsMiddleware, // Stremio doesn't show stream responses when no CORS middleware is used!
		handlers.ProxyHeaders,
		recoveryMiddleware,
		loggingMiddleware)
	s.HandleFunc("/health", healthHandler)

	// Stremio endpoints

	s.HandleFunc("/manifest.json", manifestHandler)
	s.HandleFunc("/catalog/{type}/{id}.json", catalogHandler)

	// Additional endpoints

	// Root redirects to website
	s.HandleFunc("/", rootHandler)

	srv := &http.Server{
		Addr:    addr,
		Handler: s,
		// Timeouts to avoid Slowloris attacks
		ReadTimeout:    time.Second * 5,
		WriteTimeout:   time.Second * 15,
		IdleTimeout:    time.Second * 60,
		MaxHeaderBytes: 1 * 1000, // 1 KB
	}

	log.Println("Set up server")

	stopping := false
	stoppingPtr := &stopping

	log.Printf("Starting server on %v", addr)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			if !*stoppingPtr {
				log.Fatalf("Couldn't start server: %v", err)
			} else {
				log.Fatalf("Error in srv.ListenAndServe() during server shutdown (probably context deadline expired before the server could shutdown cleanly): %v", err)
			}
		}
	}()

	// Timed logger for easier debugging with logs
	go func() {
		for {
			log.Println("...")
			time.Sleep(time.Second)
		}
	}()

	// Graceful shutdown

	c := make(chan os.Signal, 1)
	// Accept SIGINT (Ctrl+C) and SIGTERM (`docker stop`)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	sig := <-c
	log.Printf("Received signal %v, shutting down...\n", sig)
	*stoppingPtr = true
	// Create a deadline to wait for.
	// Using the same value as the server's `WriteTimeout` would be great, because this would mean that every client could finish his request as he normally could.
	// But `docker stop` only gives us 10 seconds.
	// No need to get the cancel func and defer calling it, because srv.Shutdown() will consider the timeout from the context.
	ctx, _ := context.WithTimeout(context.Background(), 9*time.Second)
	// Doesn't block if no connections, but will otherwise wait until the timeout deadline
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Error shutting down server: %v", err)
	}
	log.Println("Server shut down")
}

func createCatalogResponse(catalog string) []byte {
	buf := bytes.NewBufferString(`{"metas":[`)

	records := read(*dataDir + "/" + catalog + ".csv")
	metas := readMetas(records, *dataDir+"/metas")
	for i, meta := range metas {
		buf.WriteString(meta)
		if i < len(metas)-1 {
			buf.WriteString(",")
		}
	}

	buf.WriteString("]}")

	return buf.Bytes()
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

func readMetas(records [][]string, metasDir string) []string {
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

	var result []string
	for _, record := range records[1:] {
		imdbID := record[imdbIndex]
		// We assume that the metafetcher has been used to already write all meta JSON files for all required IMDb IDs to the directory, so we can directly read the files here via the IMDb ID + ".json", instead of going through the actual files and only read it when it matches one of our IMDb IDs.
		fileContent, err := ioutil.ReadFile(metasDir + "/" + imdbID + ".json")
		if err != nil {
			log.Printf("Couldn't read meta file for IMDb ID %v: %v", imdbID, err)
			continue
		}
		result = append(result, string(fileContent))
	}

	return result
}
