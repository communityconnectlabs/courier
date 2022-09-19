package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nyaruka/courier"
)

// NewTelReceiveHandler creates a new receive handler given the passed in text and from fields
func NewTelReceiveHandler(h courier.ChannelHandler, fromField string, bodyField string) courier.ChannelHandleFunc {
	return func(ctx context.Context, c courier.Channel, w http.ResponseWriter, r *http.Request, clog *courier.ChannelLog) ([]courier.Event, error) {
		err := r.ParseForm()
		if err != nil {
			return nil, WriteAndLogRequestError(ctx, h, c, w, r, err)
		}

		body := r.Form.Get(bodyField)
		from := r.Form.Get(fromField)
		if from == "" {
			return nil, WriteAndLogRequestError(ctx, h, c, w, r, fmt.Errorf("missing required field '%s'", fromField))
		}
		// create our URN
		urn, err := StrictTelForCountry(from, c.Country())
		if err != nil {
			return nil, WriteAndLogRequestError(ctx, h, c, w, r, err)
		}
		// build our msg
		msg := h.Server().Backend().NewIncomingMsg(c, urn, body, clog).WithReceivedOn(time.Now().UTC())
		return WriteMsgsAndResponse(ctx, h, []courier.Msg{msg}, w, r, clog)
	}
}

// NewExternalIDStatusHandler creates a new status handler given the passed in status map and fields
func NewExternalIDStatusHandler(h courier.ChannelHandler, statuses map[string]courier.MsgStatusValue, externalIDField string, statusField string) courier.ChannelHandleFunc {
	return func(ctx context.Context, c courier.Channel, w http.ResponseWriter, r *http.Request, clog *courier.ChannelLog) ([]courier.Event, error) {
		err := r.ParseForm()
		if err != nil {
			return nil, WriteAndLogRequestError(ctx, h, c, w, r, err)
		}

		externalID := r.Form.Get(externalIDField)
		if externalID == "" {
			return nil, WriteAndLogRequestError(ctx, h, c, w, r, fmt.Errorf("missing required field '%s'", externalIDField))
		}

		s := r.Form.Get(statusField)
		sValue, found := statuses[s]
		if !found {
			return nil, WriteAndLogRequestError(ctx, h, c, w, r, fmt.Errorf("unknown status value '%s'", s))
		}

		// create our status
		status := h.Server().Backend().NewMsgStatusForExternalID(c, externalID, sValue, clog)
		return WriteMsgStatusAndResponse(ctx, h, c, status, w, r)
	}
}
