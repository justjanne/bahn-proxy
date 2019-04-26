package main

import (
	"encoding/json"
	"fmt"
	"git.kuschku.de/justjanne/bahn-api"
	"log"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"
)

func returnJson(w http.ResponseWriter, data interface{}) error {
	marshalled, err := json.Marshal(data)
	if err != nil {
		return err
	}

	w.Header().Add("Content-Type", "application/json")
	if _, err := w.Write(marshalled); err != nil {
		return err
	}

	return nil
}

func main() {
	autocompleteStations := loadAutocompleteStations()

	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}

	MaxResults := 20

	http.HandleFunc("/autocomplete/", func(w http.ResponseWriter, r *http.Request) {
		if stationName := strings.TrimSpace(r.FormValue("name")); stationName != "" {
			var perfectMatch []AutocompleteStation
			var prefix []AutocompleteStation
			var contains []AutocompleteStation

			for _, station := range autocompleteStations {
				findableName := canonicalizeName(station.Name)
				if strings.EqualFold(station.Name, stationName) {
					perfectMatch = append(perfectMatch, station)
				} else if strings.EqualFold(findableName, stationName) {
					perfectMatch = append(perfectMatch, station)
				} else if strings.HasPrefix(station.Name, stationName) {
					prefix = append(prefix, station)
				} else if strings.Contains(station.Name, stationName) {
					contains = append(contains, station)
				}

				if len(perfectMatch)+len(prefix)+len(contains) >= MaxResults {
					break
				}
			}

			result := append(append(perfectMatch, prefix...), contains...)
			if err := returnJson(w, result); err != nil {
				log.Fatal(err)
				return
			}
		} else if position, err := PositionFromString(strings.TrimSpace(r.FormValue("position"))); err == nil {
			var result []AutocompleteStation
			for _, station := range autocompleteStations {
				result = append(result, AutocompleteStation{
					Id:       station.Id,
					Name:     station.Name,
					Position: station.Position,
					Distance: Distance(*station.Position, position),
				})
			}

			sort.Slice(result, func(i, j int) bool {
				return result[i].Distance < result[j].Distance
			})

			if err := returnJson(w, result[:MaxResults]); err != nil {
				log.Fatal(err)
				return
			}
		}
	})
	http.HandleFunc("/station/", func(w http.ResponseWriter, r *http.Request) {
		var err error

		_, evaId := path.Split(r.URL.Path)
		evaId = strings.TrimSpace(evaId)

		url := fmt.Sprintf("http://iris.noncd.db.de/iris-tts/timetable/station/%s", evaId)

		var response *http.Response
		if response, err = netClient.Get(url); err != nil {
			log.Fatal(err)
			return
		}

		var stations []bahn.Station
		if stations, err = bahn.StationsFromReader(response.Body); err != nil {
			log.Fatal(err)
		}
		if err = response.Body.Close(); err != nil {
			log.Fatal(err)
			return
		}
		if err = returnJson(w, stations); err != nil {
			log.Fatal(err)
			return
		}
	})
	http.HandleFunc("/timetable/", func(w http.ResponseWriter, r *http.Request) {
		var err error

		_, evaId := path.Split(r.URL.Path)
		evaId = strings.TrimSpace(evaId)

		var date time.Time
		if date, err = time.Parse(time.RFC3339, strings.TrimSpace(r.FormValue("time"))); err != nil {
			date = time.Now()
		}

		BahnFormat := "060102/15"

		url := fmt.Sprintf("http://iris.noncd.db.de/iris-tts/timetable/plan/%s/%s", evaId, date.Format(BahnFormat))

		var response *http.Response
		if response, err = netClient.Get(url); err != nil {
			log.Fatal(err)
			return
		}

		var timetable bahn.Timetable
		if timetable, err = bahn.TimetableFromReader(response.Body); err != nil {
			log.Fatal(err)
		}
		if err = response.Body.Close(); err != nil {
			log.Fatal(err)
			return
		}
		if err = returnJson(w, timetable); err != nil {
			log.Fatal(err)
			return
		}
	})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
