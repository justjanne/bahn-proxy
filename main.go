package main

import (
	"encoding/json"
	"flag"
	"git.kuschku.de/justjanne/bahn-api"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
	"log"
	"net/http"
	"os"
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

	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	if _, err := w.Write(marshalled); err != nil {
		return err
	}

	return nil
}

func measure(name string, f func()) {
	start := time.Now()
	f()
	end := time.Now()
	glog.Infof("%s took %s", name, end.Sub(start).String())
}

func main() {
	var err error

	configPath := flag.String("config", "config.yaml", "Path to config file")
	listen := flag.String("listen", ":8080", "Listen address")
	flag.Parse()

	var configFile *os.File
	if configFile, err = os.Open(*configPath); err != nil {
		panic(err)
	}

	var config Config
	if err = yaml.NewDecoder(configFile).Decode(&config); err != nil {
		panic(err)
	}

	autocompleteStations := loadAutocompleteStations()

	apiClient := bahn.ApiClient{
		IrisBaseUrl:          config.Endpoints.Iris,
		CoachSequenceBaseUrl: config.Endpoints.CoachSequence,
		HafasBaseUrl:         config.Endpoints.Hafas,
		HttpClient: &http.Client{
			Timeout: config.RequestTimeout,
		},
		Caches: []bahn.CacheBackend{
			NewMemoryCache(config.Caches.Memory.Timeout),
			NewRedisCache(
				config.Caches.Redis.Address,
				config.Caches.Redis.Password,
				config.Caches.Redis.Timeout,
			),
		},
	}

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

				if len(perfectMatch)+len(prefix)+len(contains) >= config.MaxResults {
					break
				}
			}

			result := append(append(perfectMatch, prefix...), contains...)
			if err := returnJson(w, result); err != nil {
				log.Println(err)
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

			if err := returnJson(w, result[:config.MaxResults]); err != nil {
				log.Println(err)
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
			log.Println(err)
			return
		}

		var stations []bahn.Station
		if stations, err = apiClient.Station(evaId); err != nil {
			log.Println(err)
			return
		}

		if err = returnJson(w, stations); err != nil {
			log.Println(err)
			return
		}
	})
	http.HandleFunc("/timetable/", func(w http.ResponseWriter, r *http.Request) {
		var err error

		_, rawEvaId := path.Split(r.URL.Path)
		rawEvaId = strings.TrimSpace(rawEvaId)

		var evaId int64
		if evaId, err = strconv.ParseInt(rawEvaId, 10, 64); err != nil {
			log.Println(err)
			return
		}

		var date time.Time
		if date, err = time.Parse(time.RFC3339, strings.TrimSpace(r.FormValue("time"))); err != nil {
			date = time.Now()
		}

		var data = make(map[string]InternalModel)

		measure("total", func() {

			var timetable bahn.Timetable
			measure("timetable", func() {
				if timetable, err = apiClient.Timetable(evaId, date); err != nil {
					log.Println(err)
					return
				}
				for _, stop := range timetable.Stops {
					combined := data[stop.StopId]
					combined.Timetable = stop
					data[stop.StopId] = combined
				}
			})

			var realtime bahn.Timetable
			measure("realtime", func() {
				if realtime, err = apiClient.RealtimeAll(evaId, date); err != nil {
					log.Println(err)
					return
				}
				for _, stop := range realtime.Stops {
					if combined, ok := data[stop.StopId]; ok {
						combined.Realtime = stop
						data[stop.StopId] = combined
					}
				}
			})

			measure("wing_definition", func() {
				for key, combined := range data {
					if combined.Timetable.Arrival != nil && combined.Timetable.Arrival.Wings != "" {
						if combined.WingDefinition, err = apiClient.WingDefinition(combined.Timetable.StopId, combined.Timetable.Arrival.Wings); err != nil {
							log.Println(err)
							return
						}
					} else if combined.Timetable.Departure != nil && combined.Timetable.Departure.Wings != "" {
						if combined.WingDefinition, err = apiClient.WingDefinition(combined.Timetable.StopId, combined.Timetable.Departure.Wings); err != nil {
							log.Println(err)
							return
						}
					}
					data[key] = combined
				}
			})

			/*
				measure("trainlinks", func() {
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
								log.Println(err)
								return
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
				})

				measure("hafas_messages", func() {
					for key, combined := range data {
						if combined.TrainLink != "" {
							if combined.HafasMessages, err = apiClient.HafasMessages(combined.TrainLink); err != nil {
								log.Println(err)
								return
							}
						}
						data[key] = combined
					}
				})
			*/

		})

		var result []InternalModel
		for _, element := range data {
			result = append(result, element)
		}

		if err = returnJson(w, result); err != nil {
			log.Println(err)
			return
		}
	})
	if err := http.ListenAndServe(*listen, nil); err != nil {
		log.Fatal(err)
	}
}
