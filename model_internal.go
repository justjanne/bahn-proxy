package main

import "git.kuschku.de/justjanne/bahn-api"

type InternalModel struct {
	Timetable      bahn.TimetableStop  `json:"timetable,omitempty"`
	Realtime       bahn.TimetableStop  `json:"realtime,omitempty"`
	WingDefinition bahn.WingDefinition `json:"wing_definition,omitempty"`
	HafasMessages  []bahn.HafasMessage `json:"hafas_messages,omitempty"`
	TrainLink      string              `json:"train_link,omitempty"`
}
