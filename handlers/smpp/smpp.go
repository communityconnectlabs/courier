package smpp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/nyaruka/courier"
	"github.com/nyaruka/courier/handlers"
	"github.com/nyaruka/courier/utils"
	"github.com/nyaruka/gocommon/urns"
	"net/http"
	"time"
)

func init() {
	courier.RegisterHandler(newHandler())
}

type handler struct {
	handlers.BaseHandler
}

func newHandler() courier.ChannelHandler {
	return &handler{handlers.NewBaseHandler("SMP", "SMPP")}
}

// Initialize is called by the engine once everything is loaded
func (h *handler) Initialize(s courier.Server) error {
	h.SetServer(s)
	s.AddHandlerRoute(h, http.MethodPost, "receive", h.receiveMessage)
	return nil
}

func (h *handler) SendMsg(ctx context.Context, msg courier.Msg) (courier.MsgStatus, error) {
	if msg.URN().Scheme() != urns.TelScheme {
		return nil, fmt.Errorf("wrong urn scheme for the current SMPP channel type")
	}

	payload := &moPayload{
		Channel:    msg.Channel().UUID().String(),
		ContactUrn: msg.URN().Path(),
		Text:       msg.Text(),
	}

	status := h.Backend().NewMsgStatusForID(msg.Channel(), msg.ID(), courier.MsgSent)
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("wrong urn scheme for the current SMPP channel type")
	}

	smppEndpoint := fmt.Sprintf("%s/msg", h.Server().Config().SMPPServerEndpoint)
	req, err := http.NewRequest(http.MethodPost, smppEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	rr, err := utils.MakeHTTPRequest(req)

	// record our status and log
	log := courier.NewChannelLogFromRR("Message Sent", msg.Channel(), msg.ID(), rr).WithError("Message Send Error", err)
	status.AddLog(log)
	return status, nil
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
	urn, err := urns.NewURNFromParts(urns.TelScheme, payload.ContactUrn, "", "")
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	// build our msg
	msg := h.Backend().NewIncomingMsg(channel, urn, payload.Text).WithReceivedOn(date)
	return handlers.WriteMsgsAndResponse(ctx, h, []courier.Msg{msg}, w, r)
}

type moPayload struct {
	Channel    string `json:"channel"`
	ContactUrn string `json:"contact"`
	Text       string `json:"text"`
}
