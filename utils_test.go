package courier

import (
	"context"
	"testing"

	"github.com/nyaruka/gocommon/urns"
	"github.com/stretchr/testify/assert"
)

func TestGetOptOutMessage(t *testing.T) {
	ctx := context.Context(nil)

	backend := NewMockBackend()
	channel := NewMockChannel("53e5aafa-8155-449d-9009-fcb30d54bd26", "TW", "2020", "US", map[string]interface{}{})
	contact, _ := backend.GetContact(ctx, channel, urns.URN("tel:+14133881111"), "", "")

	msg := GetOptOutMessage(channel, contact)
	assert.Equal(t, "Opt Out Message", msg)

	contact.(*mockContact).Lang = "eng"
	msg = GetOptOutMessage(channel, contact)
	assert.Equal(t, "English opt-out message", msg)

	contact.(*mockContact).Lang = "ukr"
	msg = GetOptOutMessage(channel, contact)
	assert.Equal(t, "Українське повідомлення про відписку", msg)
}
