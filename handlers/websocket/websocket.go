package websocket

import (
	"bytes"
	"context"
	"encoding/json"
	. "github.com/nyaruka/courier"
	"github.com/nyaruka/courier/handlers"
	"github.com/nyaruka/courier/utils"
	"github.com/nyaruka/gocommon/urns"
	"golang.org/x/text/language"
	"net/http"
	"strings"
	"time"
)

func init() {
	RegisterHandler(newHandler())
}

type handler struct {
	handlers.BaseHandler
}

func newHandler() ChannelHandler {
	return &handler{handlers.NewBaseHandler(ChannelType("WS"), "WebSocket")}
}

// Initialize is called by the engine once everything is loaded
func (h *handler) Initialize(s Server) error {
	h.SetServer(s)
	s.AddHandlerRoute(h, http.MethodPost, "register", h.registerUser)
	s.AddHandlerRoute(h, http.MethodPost, "receive", h.receiveMessage)
	return nil
}

// receiveMessage is our HTTP handler function for incoming messages
func (h *handler) registerUser(ctx context.Context, channel Channel, w http.ResponseWriter, r *http.Request) ([]Event, error) {
	payload := &userPayload{}
	err := handlers.DecodeAndValidateJSON(payload, r)
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	// no URN? ignore this
	if payload.URN == "" {
		return nil, handlers.WriteAndLogRequestIgnored(ctx, h, channel, w, r, "Ignoring request, no identifier")
	}

	// the list of data we will return in our response
	data := make([]interface{}, 0, 2)

	// create our URN
	urn, errURN := urns.NewURNFromParts(channel.Schemes()[0], payload.URN, "", "")
	if errURN != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, errURN)
	}

	contact, errGetContact := h.Backend().GetContact(ctx, channel, urn, "", "")
	if errGetContact != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, errGetContact)
	}

	// Getting the language in ISO3
	tag := language.MustParse(payload.Language)
	languageBase, _ := tag.Base()

	_, errLang := h.Backend().AddLanguageToContact(ctx, channel, languageBase.ISO3(), contact)
	if errLang != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, errLang)
	}

	// build our response
	data = append(data, NewEventRegisteredContactData(contact.UUID()))

	return nil, WriteDataResponse(ctx, w, http.StatusOK, "Events Handled", data)
}

// receiveMessage is our HTTP handler function for incoming messages
func (h *handler) receiveMessage(ctx context.Context, channel Channel, w http.ResponseWriter, r *http.Request) ([]Event, error) {
	payload := &msgPayload{}
	err := handlers.DecodeAndValidateJSON(payload, r)
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	// no message? ignore this
	if payload.Text == "" {
		return nil, handlers.WriteAndLogRequestIgnored(ctx, h, channel, w, r, "Ignoring request, no message")
	}

	// the list of events we deal with
	events := make([]Event, 0, 2)

	// the list of data we will return in our response
	data := make([]interface{}, 0, 2)

	urn, errURN := urns.NewURNFromParts(channel.Schemes()[0], payload.From, "", "")
	if errURN != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, errURN)
	}
	text := payload.Text

	// build our msg
	ev := h.Backend().NewIncomingMsg(channel, urn, text)
	event := h.Backend().CheckExternalIDSeen(ev)

	errMsg := h.Backend().WriteMsg(ctx, event)
	if errMsg != nil {
		return nil, errMsg
	}

	h.Backend().WriteExternalIDSeen(event)

	events = append(events, event)
	data = append(data, NewMsgReceiveData(event))

	return events, WriteDataResponse(ctx, w, http.StatusOK, "Events Handled", data)
}

func (h *handler) sendMsgPart(msg Msg, apiURL string, payload *dataPayload) (string, *ChannelLog, error) {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		log := NewChannelLog("unable to build JSON body", msg.Channel(), msg.ID(), "", "", NilStatusCode, "", "", time.Duration(0), err)
		return "", log, err
	}

	req, _ := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rr, err := utils.MakeHTTPRequest(req)

	// build our channel log
	log := NewChannelLogFromRR("Message Sent", msg.Channel(), msg.ID(), rr).WithError("Message Send Error", err)

	return "", log, nil
}

// SendMsg sends the passed in message, returning any error
func (h *handler) SendMsg(ctx context.Context, msg Msg) (MsgStatus, error) {
	address := msg.Channel().Address()

	data := &dataPayload{
		ID:          msg.ID().String(),
		Text:        msg.Text(),
		To:          msg.URN().Path(),
		ToNoPlus:    strings.Replace(msg.URN().Path(), "+", "", 1),
		From:        address,
		FromNoPlus:  strings.Replace(address, "+", "", 1),
		Channel:     strings.Replace(address, "+", "", 1),
		Metadata:    nil,
		Attachments: nil,
	}

	if len(msg.QuickReplies()) > 0 {
		quickReplies := make(map[string][]string, 0)
		quickReplies["quick_replies"] = msg.QuickReplies()
		data.Metadata = quickReplies
	}

	if len(msg.Attachments()) > 0 {
		data.Attachments = msg.Attachments()
	}

	// the status that will be written for this message
	status := h.Backend().NewMsgStatusForID(msg.Channel(), msg.ID(), MsgErrored)

	// whether we encountered any errors sending any parts
	hasError := true

	// if we have text, send that if we aren't sending it as a caption
	if msg.Text() != "" {
		externalID, log, err := h.sendMsgPart(msg, address, data)
		status.SetExternalID(externalID)
		hasError = err != nil
		status.AddLog(log)
	}

	if !hasError {
		status.SetStatus(MsgWired)
	}

	return status, nil
}

type userPayload struct {
	URN      string `json:"urn"`
	Language string `json:"language"`
}

type msgPayload struct {
	Text string `json:"text"`
	From string `json:"from"`
}

type dataPayload struct {
	ID          string              `json:"id"`
	Text        string              `json:"text"`
	To          string              `json:"to"`
	ToNoPlus    string              `json:"to_no_plus"`
	From        string              `json:"from"`
	FromNoPlus  string              `json:"from_no_plus"`
	Channel     string              `json:"channel"`
	Metadata    map[string][]string `json:"metadata"`
	Attachments []string            `json:"attachments"`
}
