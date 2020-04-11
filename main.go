package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"flag"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/doingodswork/stremio-top-movies/pkg/stremio"
)

const (
	version = "0.2.0"
)

var (
	bindAddr = flag.String("bindAddr", "localhost", `Local interface address to bind to. "localhost" only allows access from the local host. "0.0.0.0" binds to all network interfaces.`)
	port     = flag.Int("port", 8080, "Port to listen on")
	dataDir  = flag.String("dataDir", ".", "Location of the data directory. It contains CSV files with IMDb IDs and a \"metas\" subdirectory with meta JSON files")
	logLevel = flag.String("logLevel", "info", `Log level to show only logs with the given and more severe levels. Can be "trace", "debug", "info", "warn", "error", "fatal", "panic"`)
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
	responses      = make(map[string][]byte, len(catalogs))
	etags          = make(map[string]string, len(catalogs))
	cacheHeaderVal string
)

func init() {
	// Timeout for global default HTTP client (for when using `http.Get()`)
	http.DefaultClient.Timeout = 5 * time.Second

	// Configure logging (except for level, which we only know from the config which is obtained later).
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}

func main() {
	flag.Parse()

	setLogLevel(*logLevel)
	cacheAgeDuration, err := time.ParseDuration(*cacheAge)
	if err != nil {
		log.WithError(err).Fatal("Couldn't parse cacheAge")
	}
	cacheAgeSeconds := strconv.FormatFloat(math.Round(cacheAgeDuration.Seconds()), 'f', 0, 64)
	log.Debugf("Cache age will be set to %v seconds", cacheAgeSeconds)
	// `public` can be required by some proxies like CloudFlare to cache the response.
	cacheHeaderVal = "max-age=" + cacheAgeSeconds + ", public"

	// Clean input
	if strings.HasSuffix(*dataDir, "/") {
		*dataDir = strings.TrimRight(*dataDir, "/")
	}

	log.Println("Initializing catalogs...")
	for _, catalogItem := range catalogs {
		id := catalogItem.ID
		responses[id] = createCatalogResponse(id)
	}
	log.Println("Initialized catalogs")

	log.Println("Calculating ETags...")
	for _, catalogItem := range catalogs {
		id := catalogItem.ID
		hash := xxhash.Sum64(responses[id])
		etags[id] = strconv.FormatUint(hash, 16)
	}
	log.Println("Calculated ETags")

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

	addr := *bindAddr + ":" + strconv.Itoa(*port)
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
			log.Trace("...")
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
	// Create a deadline to wait for. `docker stop` gives us 10 seconds.
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
		log.Fatalf("Couldn't read file: %v", err)
	}
	csvReader := csv.NewReader(bytes.NewReader(fileBytes))
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatalf("Couldn't read CSV: %v", err)
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
		log.Fatalf("Couldn't find \"IMDb ID\" in CSV header: %v", headRecord)
	}

	var result []string
	for _, record := range records[1:] {
		imdbID := record[imdbIndex]
		// We assume that the metafetcher has been used to already write all meta JSON files for all required IMDb IDs to the directory, so we can directly read the files here via the IMDb ID + ".json", instead of going through the actual files and only read it when it matches one of our IMDb IDs.
		fileContent, err := ioutil.ReadFile(metasDir + "/" + imdbID + ".json")
		if err != nil {
			log.Errorf("Couldn't read meta file for IMDb ID %v: %v", imdbID, err)
			continue
		}
		result = append(result, string(fileContent))
	}

	return result
}

func setLogLevel(logLevel string) {
	switch logLevel {
	case "trace":
		log.SetLevel(log.TraceLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "panic":
		log.SetLevel(log.PanicLevel)
	default:
		log.WithField("logLevel", logLevel).Fatal("Unknown logLevel")
	}
}
