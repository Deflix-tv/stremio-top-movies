package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// The example code had this, but apparently it's not required and not used anywhere
// func homeHandler(w http.ResponseWriter, r *http.Request) {
// 	log.Printf("homeHandler called: %+v\n", r)
//
// 	w.Header().Set("Content-Type", "application/json")
// 	w.Write([]byte(`{"Path":"/"}`))
// }

func healthHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("healthHandler called")

	if _, err := w.Write([]byte("OK")); err != nil {
		log.Printf("Coldn't write response: %v\n", err)
	}
}

func manifestHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("manifestHandler called")

	resBody, _ := json.Marshal(manifest)

	log.Printf("Responding with: %s\n", resBody)
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(resBody); err != nil {
		log.Printf("Coldn't write response: %v\n", err)
	}
}

func catalogHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("catalogHandler called")

	params := mux.Vars(r)
	requestedType := params["type"]
	requestedID := params["id"]

	// Currently movies only
	if requestedType != "movie" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var catalogResponse []byte
	switch requestedID {
	case "imdb-top-250":
		catalogResponse = imdbTop250CatalogResponse
	case "imdb-most-popular":
		catalogResponse = imdbMostPopularCatalogResponse
	case "top-box-office-us":
		catalogResponse = boxOfficeUScatalogResponse
	case "rt-certified-fresh":
		catalogResponse = rtCertifiedFreshCatalogResponse
	case "academy-awards-winners":
		catalogResponse = academyAwardsCatalogResponse
	case "palme-dor-winners":
		catalogResponse = palmeDorCatalogResponse
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}

	log.Printf("Responding with: %s\n", catalogResponse)
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(catalogResponse); err != nil {
		log.Println("Coldn't write response:", err)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("rootHandler called")

	log.Printf("Responding with redirect to %v\n", redirectURL)
	w.Header().Set("Location", redirectURL)
	w.WriteHeader(http.StatusMovedPermanently)
}