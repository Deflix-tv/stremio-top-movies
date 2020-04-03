package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// The example code had this, but apparently it's not required and not used anywhere
// func homeHandler(w http.ResponseWriter, r *http.Request) {
// 	log.Printf("homeHandler called: %+v\n", r)
//
// 	w.Header().Set("Content-Type", "application/json")
// 	w.Write([]byte(`{"Path":"/"}`))
// }

func healthHandler(w http.ResponseWriter, r *http.Request) {
	log.Trace("healthHandler called")

	if _, err := w.Write([]byte("OK")); err != nil {
		log.Errorf("Coldn't write response: %v\n", err)
	}
}

func manifestHandler(w http.ResponseWriter, r *http.Request) {
	log.Trace("manifestHandler called")

	resBody, _ := json.Marshal(manifest)

	log.Debugf("Responding with: %s\n", resBody)
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(resBody); err != nil {
		log.Errorf("Coldn't write response: %v\n", err)
	}
}

func catalogHandler(w http.ResponseWriter, r *http.Request) {
	log.Trace("catalogHandler called")

	params := mux.Vars(r)
	requestedType := params["type"]
	requestedID := params["id"]

	// Currently movies only
	if requestedType != "movie" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ifNoneMatch := r.Header.Get("If-None-Match")

	var catalogResponse []byte
	var etag string
	switch requestedID {
	case "imdb-top-250":
		etag = imdbTop250CatalogResponseEtag
		if ifNoneMatch != etag {
			catalogResponse = imdbTop250CatalogResponse
		}
	case "imdb-most-popular":
		etag = imdbMostPopularCatalogResponseEtag
		if ifNoneMatch != etag {
			catalogResponse = imdbMostPopularCatalogResponse
		}
	case "top-box-office-us":
		etag = boxOfficeUScatalogResponseEtag
		if ifNoneMatch != etag {
			catalogResponse = boxOfficeUScatalogResponse
		}
	case "rt-certified-fresh":
		etag = rtCertifiedFreshCatalogResponseEtag
		if ifNoneMatch != etag {
			catalogResponse = rtCertifiedFreshCatalogResponse
		}
	case "academy-awards-winners":
		etag = academyAwardsCatalogResponseEtag
		if ifNoneMatch != etag {
			catalogResponse = academyAwardsCatalogResponse
		}
	case "palme-dor-winners":
		etag = palmeDorCatalogResponseEtag
		if ifNoneMatch != etag {
			catalogResponse = palmeDorCatalogResponse
		}
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}

	notModified := false
	if ifNoneMatch == "*" {
		log.Debug("If-None-Match is \"*\", responding with 304")
		notModified = true
	} else if len(catalogResponse) == 0 {
		log.WithField("ETag", ifNoneMatch).Debug("ETag matches, responding with 304")
		notModified = true
	}
	if notModified {
		w.Header().Set("Cache-Control", cacheHeaderVal) // Required according to https://tools.ietf.org/html/rfc7232#section-4.1
		w.Header().Set("ETag", etag)                    // We set it to make sure a client doesn't overwrite its cached ETag with an empty string or so.
		w.WriteHeader(http.StatusNotModified)
		return
	}

	log.Debugf("Responding with: %s\n", catalogResponse)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", cacheHeaderVal)
	w.Header().Set("ETag", etag)
	if _, err := w.Write(catalogResponse); err != nil {
		log.Errorf("Coldn't write response: %v\n", err)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	log.Trace("rootHandler called")

	log.Debugf("Responding with redirect to %v\n", redirectURL)
	w.Header().Set("Location", redirectURL)
	w.WriteHeader(http.StatusMovedPermanently)
}
