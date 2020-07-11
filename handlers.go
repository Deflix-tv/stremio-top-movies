package main

import (
	"github.com/deflix-tv/go-stremio"
)

func movieHandler(id string, userData interface{}) ([]stremio.MetaPreviewItem, error) {
	// Existing catalog IDs only
	found := false
	for _, catalogItem := range catalogs {
		if id == catalogItem.ID {
			found = true
			break
		}
	}
	if !found {
		return nil, stremio.NotFound
	}

	// Return the prepared response
	return responses[id], nil
}
