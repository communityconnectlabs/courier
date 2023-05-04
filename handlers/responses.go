package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/nyaruka/courier"
)

// ResponseWriter interace with response methods for success responses
type ResponseWriter interface {
	Backend() courier.Backend
	WriteStatusSuccessResponse(ctx context.Context, w http.ResponseWriter, r *http.Request, statuses []courier.MsgStatus) error
	WriteMsgSuccessResponse(ctx context.Context, w http.ResponseWriter, r *http.Request, msgs []courier.Msg) error
	WriteRequestError(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) error
	WriteRequestIgnored(ctx context.Context, w http.ResponseWriter, r *http.Request, msg string) error
}

// WriteMsgsAndResponse writes the passed in message to our backend
func WriteMsgsAndResponse(ctx context.Context, h ResponseWriter, msgs []courier.Msg, w http.ResponseWriter, r *http.Request) ([]courier.Event, error) {
	events := make([]courier.Event, len(msgs), len(msgs))
	for i, m := range msgs {
		err := h.Backend().WriteMsg(ctx, m)
		if err != nil {
			return nil, err
		}
		events[i] = m

		if m.Channel().ChannelType().String() == "MGA" {
			_ = h.Backend().WriteSMPPLog(ctx, &courier.SMPPLog{
				ChannelID: m.Channel().ID(),
				MsgID:     m.ID(),
				Status:    courier.MsgHandled,
				CreatedOn: time.Now(),
			})
		}

	}

	return events, h.WriteMsgSuccessResponse(ctx, w, r, msgs)
}

func WriteMsgAttachmentAndResponse(ctx context.Context, h ResponseWriter, msg *courier.Msg, a *courier.MsgAttachment, w http.ResponseWriter, r *http.Request) ([]courier.Event, error) {
	events := make([]courier.Event, 1, 1)
	events[0] = *msg

	err := h.Backend().WriteMsgAttachment(ctx, (*msg).Channel(), a)
	if err != nil {
		return nil, err
	}

	return events, h.WriteMsgSuccessResponse(ctx, w, r, []courier.Msg{*msg})
}

// WriteMsgStatusAndResponse write the passed in status to our backend
func WriteMsgStatusAndResponse(ctx context.Context, h ResponseWriter, channel courier.Channel, status courier.MsgStatus, w http.ResponseWriter, r *http.Request) ([]courier.Event, error) {
	err := h.Backend().WriteMsgStatus(ctx, status)
	if err == courier.ErrMsgNotFound {
		return nil, WriteAndLogRequestIgnored(ctx, h, channel, w, r, "msg not found, ignored")
	}

	if err != nil {
		return nil, err
	}

	if channel.ChannelType().String() == "MGA" {
		_ = h.Backend().WriteSMPPLog(ctx, &courier.SMPPLog{
			ChannelID: status.ChannelID(),
			MsgID:     status.ID(),
			Status:    status.Status(),
			CreatedOn: time.Now(),
		})
	}

	return []courier.Event{status}, h.WriteStatusSuccessResponse(ctx, w, r, []courier.MsgStatus{status})
}

// WriteAndLogRequestError logs the passed in error and writes the response to the response writer
func WriteAndLogRequestError(ctx context.Context, h ResponseWriter, channel courier.Channel, w http.ResponseWriter, r *http.Request, err error) error {
	courier.LogRequestError(r, channel, err)
	return h.WriteRequestError(ctx, w, r, err)
}

// WriteAndLogRequestIgnored logs that the passed in request was ignored and writes the response to the response writer
func WriteAndLogRequestIgnored(ctx context.Context, h ResponseWriter, channel courier.Channel, w http.ResponseWriter, r *http.Request, details string) error {
	courier.LogRequestIgnored(r, channel, details)
	return h.WriteRequestIgnored(ctx, w, r, details)
}
