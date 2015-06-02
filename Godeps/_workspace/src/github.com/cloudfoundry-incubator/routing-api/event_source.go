package routing_api

import (
	"encoding/json"

	"github.com/cloudfoundry-incubator/routing-api/db"
	"github.com/cloudfoundry-incubator/routing-api/trace"
	"github.com/vito/go-sse/sse"
	"time"
)

type EventSource interface {
	Next() (Event, error)
	Close() error
}

type RawEventSource interface {
	Next() (sse.Event, error)
	Close() error
}

type eventSource struct {
	rawEventSource RawEventSource
}

type Event struct {
	Route  db.Route
	Action string
}

func NewEventSource(raw RawEventSource) EventSource {
	return &eventSource{
		rawEventSource: raw,
	}
}

func (e *eventSource) Next() (Event, error) {
	rawEvent, err := e.rawEventSource.Next()
	if err != nil {
		return Event{}, err
	}

	dumpSSEEvent(rawEvent)

	event, err := convertRawEvent(rawEvent)
	if err != nil {
		return Event{}, err
	}

	return event, nil
}

func (e *eventSource) Close() error {
	err := e.rawEventSource.Close()
	if err != nil {
		return err
	}

	return nil
}

func convertRawEvent(event sse.Event) (Event, error) {
	var route db.Route

	err := json.Unmarshal(event.Data, &route)
	if err != nil {
		return Event{}, err
	}

	return Event{Action: event.Name, Route: route}, nil
}

func dumpSSEEvent(event sse.Event) {
	eventJson, err := json.Marshal(event)
	if err != nil {
		trace.Logger.Printf("Error dumping event\n%s\n", err)
	} else {
		trace.Logger.Printf("\n%s [%s]\n%s\n", "EVENT:", time.Now().Format(time.RFC3339), trace.Sanitize(string(eventJson)))
	}
}
