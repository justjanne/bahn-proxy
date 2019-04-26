package main

import (
	"encoding/json"
	"log"
	"os"
	"regexp"
	"strings"
)

type AutocompleteStation struct {
	Id       int64   `json:"stop_id"`
	Name     string  `json:"stop_name"`
	*Position
	Distance float64 `json:"stop_distance,omitempty"`
}

func loadAutocompleteStations() []AutocompleteStation {
	var file *os.File
	var err error
	if file, err = os.Open("assets/stops.json"); err != nil {
		log.Fatal(err)
	}
	var autocompleteStations []AutocompleteStation
	if err = json.NewDecoder(file).Decode(&autocompleteStations); err != nil {
		log.Fatal(err)
	}
	if err = file.Close(); err != nil {
		log.Fatal(err)
	}
	return autocompleteStations
}

func canonicalizeName(stationName string) string {
	additionalRegex := regexp.MustCompile("\\([^(]*\\)")
	spaceRegex := regexp.MustCompile("  +")
	stationName = additionalRegex.ReplaceAllString(stationName, "")
	stationName = spaceRegex.ReplaceAllString(stationName, " ")
	stationName = strings.TrimSpace(stationName)
	stationName = strings.TrimSuffix(stationName, " Hbf")
	return stationName
}
