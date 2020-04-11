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

	// Existing catalog IDs only
	found := false
	for _, catalogItem := range catalogs {
		if requestedID == catalogItem.ID {
			found = true
			break
		}
	}
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Set a response if the client sends no or a different value
	ifNoneMatch := r.Header.Get("If-None-Match")
	etag := etags[requestedID]
	var response []byte
	modified := false
	if ifNoneMatch == "*" {
		log.Debug("If-None-Match is \"*\", responding with 304")
		// Keep modified `false`
	} else if ifNoneMatch != etag {
		log.WithField("If-None-Match", ifNoneMatch).WithField("ETag", etag).Debug("If-None-Match != ETag")
		response = responses[requestedID]
		modified = true
	} else {
		log.WithField("ETag", etag).Debug("ETag matches, responding with 304")
	}

	if !modified {
		w.Header().Set("Cache-Control", cacheHeaderVal) // Required according to https://tools.ietf.org/html/rfc7232#section-4.1
		w.Header().Set("ETag", etag)                    // We set it to make sure a client doesn't overwrite its cached ETag with an empty string or so.
		w.WriteHeader(http.StatusNotModified)
		return
	}
	log.Debugf("Responding with: %s\n", response)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", cacheHeaderVal)
	w.Header().Set("ETag", etag)
	if _, err := w.Write(response); err != nil {
		log.Errorf("Coldn't write response: %v\n", err)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	log.Trace("rootHandler called")

	log.Debugf("Responding with redirect to %v\n", redirectURL)
	w.Header().Set("Location", redirectURL)
	w.WriteHeader(http.StatusMovedPermanently)
}
