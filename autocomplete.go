package main

import (
	"encoding/json"
	"log"
	"os"
	"regexp"
	"strings"
)

type AutocompleteStation struct {
	Id                int64  `json:"stop_id"`
	Name              string `json:"stop_name"`
	CanonicalizedName string `json:"-"`
	FindableName      string `json:"-"`
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
	for i := range autocompleteStations {
		autocompleteStations[i].CanonicalizedName = canonicalizeName(autocompleteStations[i].Name)
		autocompleteStations[i].FindableName = findableName(autocompleteStations[i].Name)
	}
	if err = file.Close(); err != nil {
		log.Fatal(err)
	}
	return autocompleteStations
}

func findableName(stationName string) string {
	additionalRegex := regexp.MustCompile("\\([^(]*\\)")
	spaceRegex := regexp.MustCompile("  +")

	stationName = canonicalizeName(stationName)
	stationName = additionalRegex.ReplaceAllString(stationName, "")
	stationName = spaceRegex.ReplaceAllString(stationName, " ")
	stationName = strings.TrimSpace(stationName)
	stationName = strings.TrimSuffix(stationName, " hbf")
	return stationName
}

func canonicalizeName(stationName string) string {
	stationName = strings.ToLower(stationName)
	stationName = strings.TrimSpace(stationName)
	stationName = strings.ReplaceAll(stationName, "ü", "ue")
	stationName = strings.ReplaceAll(stationName, "ä", "ae")
	stationName = strings.ReplaceAll(stationName, "ö", "oe")
	stationName = strings.ReplaceAll(stationName, "ß", "ss")
	return stationName
}
