package main

import (
	"git.kuschku.de/justjanne/bahn-api"
)

func MergeTimetableStop(data *bahn.TimetableStop, realtime *bahn.TimetableStop) {
	if data == nil || realtime == nil {
		return
	}

	data.Messages = realtime.Messages
	MergeTripLabel(&data.TripLabel, &realtime.TripLabel)
	MergeTimetableStop(data.Ref, realtime.Ref)
	MergeEvent(data.Arrival, realtime.Arrival)
	MergeEvent(data.Departure, realtime.Departure)
	data.HistoricDelays = realtime.HistoricDelays
	data.HistoricPlatformChanges = realtime.HistoricPlatformChanges
	data.Connections = realtime.Connections
}

func MergeTripLabel(data *bahn.TripLabel, realtime *bahn.TripLabel) {
	if data == nil || realtime == nil {
		return
	}

	data.Messages = realtime.Messages
}

func MergeEvent(data *bahn.Event, realtime *bahn.Event) {
	if data == nil || realtime == nil {
		return
	}

	data.Messages = realtime.Messages
	data.ChangedPlatform = realtime.ChangedPlatform
	data.ChangedTime = realtime.ChangedTime
	data.ChangedPath = realtime.ChangedPath
	data.ChangedDestination = realtime.ChangedDestination
	data.ChangedStatus = realtime.ChangedStatus
}
