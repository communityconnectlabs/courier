package handlers_test

import (
	"context"
	"github.com/nyaruka/courier"
	"github.com/nyaruka/courier/handlers"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestNewTelReceiveHandler(t *testing.T) {
	server := courier.NewServer(courier.NewConfig(), courier.NewMockBackend())
	w := httptest.NewRecorder()
	channel := courier.NewMockChannel("e4bb1578-29da-4fa5-a214-9da19dd24230", "DM", "2020", "US", map[string]interface{}{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	r := httptest.NewRequest("GET", "https://example.com", nil)
	h := handlers.NewBaseHandler("AC", "Arabia Cell")
	h.SetServer(server)
	receiver := handlers.NewTelReceiveHandler(&h, "from", "body")
	events, err := receiver(ctx, channel, w, r)
	assert.NoError(t, err)

	assert.Equal(t, 0, len(events))
	r.Form = url.Values{"from": []string{"+2348067886565"}, "body": []string{"this is from a friend"}}
	events, err = receiver(ctx, channel, w, r)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(events))
}

func TestNewExternalIDStatusHandler(t *testing.T) {
	server := courier.NewServer(courier.NewConfig(), courier.NewMockBackend())
	w := httptest.NewRecorder()
	channel := courier.NewMockChannel("e4bb1578-29da-4fa5-a214-9da19dd24230", "DM", "2020", "US", map[string]interface{}{})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	r := httptest.NewRequest("GET", "https://example.com", nil)
	h := handlers.NewBaseHandler("AC", "Arabia Cell")
	h.SetServer(server)

	statuses := map[string]courier.MsgStatusValue{"failed": courier.MsgFailed, "sent": courier.MsgSent}
	statusHandler := handlers.NewExternalIDStatusHandler(&h, statuses, "extID", "status")

	events, err := statusHandler(ctx, channel, w, r)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(events))

	r.Form = url.Values{"extID": []string{"100"}, "status": []string{"failed"}}
	events, err = statusHandler(ctx, channel, w, r)
	assert.NoError(t, err)
	event := events[0]
	assert.Equal(t, 1, len(events))
	e := newEvent(event)

	assert.Equal(t, "100", e.ExternalID())
	assert.Equal(t, courier.MsgFailed, e.Status())

	// using status not defined will result in zero events
	r.Form = url.Values{"extID": []string{"100"}, "status": []string{"notFound"}}
	events, err = statusHandler(ctx, channel, w, r)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(events))
}

func newEvent(event courier.Event) mockEvent {
	return event.(mockEvent)
}

type mockEvent interface {
	ExternalID() string
	Status() courier.MsgStatusValue
}
