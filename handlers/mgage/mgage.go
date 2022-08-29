package mgage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/nyaruka/courier"
	"github.com/nyaruka/courier/handlers"
	"github.com/nyaruka/courier/utils"
	"github.com/nyaruka/gocommon/gsm7"
	"github.com/nyaruka/gocommon/urns"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"time"
)

func init() {
	courier.RegisterHandler(newHandler())
}

type handler struct {
	handlers.BaseHandler
}

func newHandler() courier.ChannelHandler {
	return &handler{handlers.NewBaseHandler("MGA", "mGage")}
}

// Initialize is called by the engine once everything is loaded
func (h *handler) Initialize(s courier.Server) error {
	h.SetServer(s)
	s.AddHandlerRoute(h, http.MethodPost, "receive", h.receiveMessage)
	s.AddHandlerRoute(h, http.MethodPost, "status", h.receiveStatus)
	return nil
}

func (h *handler) SendMsg(ctx context.Context, msg courier.Msg) (courier.MsgStatus, error) {
	if msg.URN().Scheme() != urns.TelScheme {
		return nil, fmt.Errorf("wrong urn scheme for the current mGage channel type")
	}

	msgEncoding := GSM7
	isGSM := gsm7.IsValid(msg.Text())
	if !isGSM {
		msgEncoding = UCS2
	}

	if h.shouldSplit(msg.Text(), msgEncoding) {
		status := h.Backend().NewMsgStatusForID(msg.Channel(), msg.ID(), courier.MsgWired)
		parts := h.encodeSplit(msg.Text(), msgEncoding)
		partsLength := len(parts)
		for index, part := range parts {
			payload := &moPayload{
				ID:       msg.ID().String(),
				Sender:   msg.Channel().Address(),
				Receiver: msg.URN().Path(),
				Text:     part,
				Encoding: string(msgEncoding),
				PartNum:  index + 1,
				Parts:    partsLength,
			}

			rr, err := h.sendToSMPP(payload)
			log := courier.NewChannelLogFromRR("Message Sent", msg.Channel(), msg.ID(), rr).WithError("Message Send Error", err)
			status.AddLog(log)
		}
		return status, nil
	} else {
		payload := &moPayload{
			ID:       msg.ID().String(),
			Sender:   msg.Channel().Address(),
			Receiver: msg.URN().Path(),
			Text:     msg.Text(),
			Encoding: string(msgEncoding),
			PartNum:  1,
			Parts:    1,
		}
		rr, err := h.sendToSMPP(payload)
		status := h.Backend().NewMsgStatusForID(msg.Channel(), msg.ID(), courier.MsgWired)
		log := courier.NewChannelLogFromRR("Message Sent", msg.Channel(), msg.ID(), rr).WithError("Message Send Error", err)
		status.AddLog(log)
		return status, nil
	}
}

func (h *handler) receiveMessage(ctx context.Context, channel courier.Channel, w http.ResponseWriter, r *http.Request) ([]courier.Event, error) {
	payload := &moPayload{}
	err := handlers.DecodeAndValidateJSON(payload, r)
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	// create our date from the timestamp
	date := time.Now().UTC()

	// create our URN
	urn, err := urns.NewURNFromParts(urns.TelScheme, payload.Sender, "", "")
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	// build our msg
	msg := h.Backend().NewIncomingMsg(channel, urn, payload.Text).WithReceivedOn(date)
	return handlers.WriteMsgsAndResponse(ctx, h, []courier.Msg{msg}, w, r)
}

func (h *handler) receiveStatus(ctx context.Context, channel courier.Channel, w http.ResponseWriter, r *http.Request) ([]courier.Event, error) {
	// if the message id was passed explicitly, use that
	var status courier.MsgStatus
	idString := r.URL.Query().Get("id")
	if idString != "" {
		msgID, err := strconv.ParseInt(idString, 10, 64)
		if err != nil {
			logrus.WithError(err).WithField("id", idString).Error("error converting mGage callback id to integer")
		} else {
			status = h.Backend().NewMsgStatusForID(channel, courier.NewMsgID(msgID), "")
		}
	}

	// if we have no status, then build it from the external id
	if status == nil {
		status = h.Backend().NewMsgStatusForExternalID(channel, "", "")
	}

	if status.ID() == courier.NilMsgID && status.ExternalID() == "" {
		return nil, handlers.WriteAndLogRequestIgnored(ctx, h, channel, w, r, "no msg status, ignoring")
	}

	return handlers.WriteMsgStatusAndResponse(ctx, h, channel, status, w, r)
}

func (h *handler) shouldSplit(text string, encoding MsgEncoding) (shouldSplit bool) {
	if encoding == UCS2 {
		return uint(len(text)*2) > SmMsgLen
	}
	return uint(len(text)) > SmMsgLen
}

func (h *handler) encodeSplit(text string, encoding MsgEncoding) []string {
	var allSeg []string
	var runeSlice = []rune(text)
	var octetLimit = 134
	var hextetLimit = int(octetLimit / 2) // round down

	limit := octetLimit
	if encoding != GSM7 {
		limit = hextetLimit
	}

	fr, to := 0, limit
	for fr < len(runeSlice) {
		if to > len(runeSlice) {
			to = len(runeSlice)
		}
		seg := string(runeSlice[fr:to])
		allSeg = append(allSeg, seg)
		fr, to = to, to+limit
	}

	return allSeg
}

func (h *handler) sendToSMPP(data interface{}) (*utils.RequestResponse, error) {
	smppEndpoint := fmt.Sprintf("%s/send-msg", h.Server().Config().SMPPServerEndpoint)
	jsonBody, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("wrong urn scheme for the current mGage channel type")
	}
	req, err := http.NewRequest(http.MethodPost, smppEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", h.Server().Config().SMPPServerToken)
	return utils.MakeHTTPRequest(req)
}

type MsgEncoding string

const (
	GSM7     MsgEncoding = "GSM7"
	UCS2     MsgEncoding = "UCS2"
	SmMsgLen uint        = 140
)

type moPayload struct {
	ID       string `json:"id,omitempty"`
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Encoding string `json:"encoding"`
	Text     string `json:"text"`
	Parts    int    `json:"parts"`
	PartNum  int    `json:"part_num"`
}
