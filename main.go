package main

import (
	"encoding/json"
	"fmt"
	"git.kuschku.de/justjanne/bahn-api"
	"log"
	"net/http"
	"path"
	"sort"
	"strconv"
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

	apiClient := bahn.ApiClient{
		IrisBaseUrl:          "http://iris.noncd.db.de/iris-tts",
		CoachSequenceBaseUrl: "https://www.apps-bahn.de/wr/wagenreihung/1.0",
		HafasBaseUrl:         "https://reiseauskunft.bahn.de/bin",
		HttpClient: &http.Client{
			Timeout: time.Second * 10,
		},
		Caches: []bahn.CacheBackend{
			NewMemoryCache(5 * time.Minute),
		},
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

		_, rawEvaId := path.Split(r.URL.Path)
		rawEvaId = strings.TrimSpace(rawEvaId)

		var evaId int64
		if evaId, err = strconv.ParseInt(rawEvaId, 10, 64); err != nil {
			log.Fatal(err)
			return
		}

		var stations []bahn.Station
		if stations, err = apiClient.Station(evaId); err != nil {
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

		_, rawEvaId := path.Split(r.URL.Path)
		rawEvaId = strings.TrimSpace(rawEvaId)

		var evaId int64
		if evaId, err = strconv.ParseInt(rawEvaId, 10, 64); err != nil {
			log.Fatal(err)
			return
		}

		var date time.Time
		if date, err = time.Parse(time.RFC3339, strings.TrimSpace(r.FormValue("time"))); err != nil {
			date = time.Now()
		}

		var data = make(map[string]InternalModel)

		var timetable bahn.Timetable
		if timetable, err = apiClient.Timetable(evaId, date); err != nil {
			log.Fatal(err)
			return
		}
		for _, stop := range timetable.Stops {
			combined := data[stop.StopId]
			combined.Timetable = stop
			data[stop.StopId] = combined
		}

		var realtime bahn.Timetable
		if realtime, err = apiClient.RealtimeAll(evaId, date); err != nil {
			log.Fatal(err)
			return
		}
		for _, stop := range realtime.Stops {
			if combined, ok := data[stop.StopId]; ok {
				combined.Realtime = stop
				data[stop.StopId] = combined
			}
		}

		for key, combined := range data {
			if combined.Timetable.Arrival != nil && combined.Timetable.Arrival.Wings != "" {
				if combined.WingDefinition, err = apiClient.WingDefinition(combined.Timetable.StopId, combined.Timetable.Arrival.Wings); err != nil {
					log.Fatal(err)
				}
			} else if combined.Timetable.Departure != nil && combined.Timetable.Departure.Wings != "" {
				if combined.WingDefinition, err = apiClient.WingDefinition(combined.Timetable.StopId, combined.Timetable.Departure.Wings); err != nil {
					log.Fatal(err)
				}
			}
			data[key] = combined
		}

		for key, combined := range data {
			var moment time.Time
			if combined.Timetable.Departure != nil && combined.Timetable.Departure.PlannedTime != nil {
				moment = *combined.Timetable.Departure.PlannedTime
			} else if combined.Timetable.Arrival != nil && combined.Timetable.Arrival.PlannedTime != nil {
				moment = *combined.Timetable.Arrival.PlannedTime
			}

			if !moment.IsZero() {
				searchQuery := fmt.Sprintf("%s %s", combined.Timetable.TripLabel.TripCategory, combined.Timetable.TripLabel.TripNumber)
				var suggestions []bahn.Suggestion
				if suggestions, err = apiClient.Suggestions(searchQuery, moment); err != nil {
					log.Fatal(err)
				}
				var targetStation = timetable.Station
				if combined.Timetable.Departure != nil {
					if combined.Timetable.Departure.PlannedDestination != "" {
						targetStation = combined.Timetable.Departure.PlannedDestination
					} else if len(combined.Timetable.Departure.PlannedPath) > 0 {
						targetStation = combined.Timetable.Departure.PlannedPath[len(combined.Timetable.Departure.PlannedPath)-1]
					}
				}
				var sourceStation = timetable.Station
				if combined.Timetable.Arrival != nil {
					if combined.Timetable.Arrival.PlannedDestination != "" {
						sourceStation = combined.Timetable.Arrival.PlannedDestination
					} else if len(combined.Timetable.Arrival.PlannedPath) > 0 {
						sourceStation = combined.Timetable.Arrival.PlannedPath[0]
					}
				}
				for _, suggestion := range suggestions {
					if targetStation == suggestion.ArrivalStation || sourceStation == suggestion.DepartureStation {
						combined.TrainLink = suggestion.TrainLink
					}
				}
			}
			data[key] = combined
		}

		for key, combined := range data {
			if combined.TrainLink != "" {
				if combined.HafasMessages, err = apiClient.HafasMessages(combined.TrainLink); err != nil {
					log.Fatal(err)
				}
			}
			data[key] = combined
		}

		if err = returnJson(w, data); err != nil {
			log.Fatal(err)
			return
		}
	})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
