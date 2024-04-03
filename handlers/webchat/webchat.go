package webchat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	. "github.com/nyaruka/courier"
	"github.com/nyaruka/courier/backends/rapidpro"
	"github.com/nyaruka/courier/handlers"
	"github.com/nyaruka/courier/utils"
	"github.com/nyaruka/gocommon/urns"
	"golang.org/x/text/language"
)

func init() {
	RegisterHandler(newHandler())
}

type handler struct {
	handlers.BaseHandler
}

func newHandler() ChannelHandler {
	return &handler{handlers.NewBaseHandler(ChannelType("WCH"), "WebChat")}
}

// Initialize is called by the engine once everything is loaded
func (h *handler) Initialize(s Server) error {
	h.SetServer(s)
	s.AddHandlerRoute(h, http.MethodPost, "register", h.registerUser)
	s.AddHandlerRoute(h, http.MethodPost, "receive", h.receiveMessage)
	s.AddHandlerRoute(h, http.MethodPost, "history", h.chatHistory)
	s.AddHandlerRoute(h, http.MethodPost, "feedback", h.receiveFeedback)
	s.AddHandlerRoute(h, http.MethodPost, "set-agent", h.setAgent)
	return nil
}

// registerUser is our HTTP handler function for register websocket contacts
func (h *handler) registerUser(ctx context.Context, channel Channel, w http.ResponseWriter, r *http.Request) ([]Event, error) {
	payload := &userPayload{}
	err := handlers.DecodeAndValidateJSON(payload, r)
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	// the list of data we will return in our response
	data := make([]interface{}, 0, 2)

	var urn urns.URN
	var errURN error
	var userToken string
	if payload.UserToken == "" {
		// no URN? ignore this
		if payload.URN == "" {
			return nil, handlers.WriteAndLogRequestIgnored(ctx, h, channel, w, r, "Ignoring request, no identifier")
		}

		// create our URN
		urn, errURN = urns.NewURNFromParts(channel.Schemes()[0], payload.URN, "", "")
		userToken = CreateToken(urn.String(), h.Server().Config().WebChatServerSecret)
	} else {
		// decode token
		userToken = payload.UserToken
		urnString, err := urnFromToken(payload.UserToken, h.Server().Config().WebChatServerSecret)
		if err == nil {
			// get our urn from token
			urn, errURN = urns.Parse(urnString)
		}
	}
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

	responseErrors := make([]string, 0)
	if payload.Extra != nil {
		for fieldName, FieldValue := range payload.Extra {
			sanitazedValue := fmt.Sprintf("%v", FieldValue)
			_, err = h.Backend().SetContactCustomField(ctx, contact, fieldName, sanitazedValue)
			if err != nil {
				responseErrors = append(responseErrors, fmt.Sprintf("field %s not found on the org", fieldName))
			}
		}
	}

	// build our response
	data = append(data, NewEventRegisteredContactData(contact.UUID(), userToken, urn.Path(), responseErrors))

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
	if payload.Text == "" && payload.AttachmentURL == "" {
		return nil, handlers.WriteAndLogRequestIgnored(ctx, h, channel, w, r, "Ignoring request, no message or no attachment")
	}

	urn, errURN := urns.NewURNFromParts(channel.Schemes()[0], payload.From, "", "")
	if errURN != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, errURN)
	}
	text := payload.Text

	msg := h.Backend().NewIncomingMsg(channel, urn, text)

	if payload.AttachmentURL != "" {
		msg.WithAttachment(payload.AttachmentURL)
	}

	return handlers.WriteMsgsAndResponse(ctx, h, []Msg{msg}, w, r)
}

func (h *handler) chatHistory(ctx context.Context, channel Channel, w http.ResponseWriter, r *http.Request) ([]Event, error) {
	payload := &userPayload{}
	err := handlers.DecodeAndValidateJSON(payload, r)
	if err != nil || payload.UserToken == "" {
		return nil, handlers.WriteAndLogRequestIgnored(ctx, h, channel, w, r, "Ignoring request, no contact token")
	}

	urnString, err := urnFromToken(payload.UserToken, h.Server().Config().WebChatServerSecret)
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	urn, errURN := urns.Parse(urnString)
	if errURN != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, errURN)
	}

	contact, errGetContact := h.Backend().GetContact(ctx, channel, urn, "", "")
	if errGetContact != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, errGetContact)
	}

	msgs, err := h.Backend().GetContactMessages(channel, contact)
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	// the list of data we will return in our response
	responseMsgs := make([]*webChatMessagePayload, 0)
	for _, msg := range msgs {
		origin := "user"
		msgDirection := msg.(*rapidpro.DBMsg).Direction_
		if msgDirection == "O" {
			origin = "ws"
		}
		responseMsg := &webChatMessagePayload{
			Message:     msg.Text(),
			Origin:      origin,
			Metadata:    nil,
			Attachments: nil,
		}

		metadata := make(map[string]interface{}, 0)

		if len(msg.QuickReplies()) > 0 {
			buildQuickReplies := make([]string, 0)
			for _, item := range msg.QuickReplies() {
				item = strings.ReplaceAll(item, "\\/", "/")
				item = strings.ReplaceAll(item, "\\\"", "\"")
				item = strings.ReplaceAll(item, "\\\\", "\\")
				buildQuickReplies = append(buildQuickReplies, item)
			}
			metadata["quick_replies"] = buildQuickReplies
		}

		if len(msg.Attachments()) > 0 {
			responseMsg.Attachments = msg.Attachments()
		}

		if msg.ReceiveAttachment() != "" {
			metadata["receive_attachment"] = msg.ReceiveAttachment()
		}

		if msg.SharingConfig() != nil {
			metadata["sharing_config"] = msg.SharingConfig()
		}

		if msg.Topic() == "feedback" {
			metadata["feedback"] = true
		}
		responseMsg.Metadata = metadata
		responseMsgs = append(responseMsgs, responseMsg)
	}
	return nil, WriteDataResponse(ctx, w, http.StatusOK, "Events Handled", []interface{}{responseMsgs})
}

func (h *handler) receiveFeedback(ctx context.Context, channel Channel, w http.ResponseWriter, r *http.Request) ([]Event, error) {
	payload := &feedbackResponsePayload{}
	err := handlers.DecodeAndValidateJSON(payload, r)
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	// no message? ignore this
	if payload.FeedbackComment == "" {
		return nil, handlers.WriteAndLogRequestIgnored(ctx, h, channel, w, r, "Ignoring request, no message or no attachment")
	}

	urn, errURN := urns.NewURNFromParts(channel.Schemes()[0], payload.From, "", "")
	if errURN != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, errURN)
	}

	body := &feedbackResponseBodyPayload{
		feedbackResponseAnswerPayload{
			StarRatingAnswer: payload.FeedbackRate,
			CommentAnswer:    payload.FeedbackComment,
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, errURN)
	}
	msg := h.Backend().NewIncomingMsg(channel, urn, string(data))
	msg.WithMetadata([]byte(`{"topic":"feedback"}`))

	return handlers.WriteMsgsAndResponse(ctx, h, []Msg{msg}, w, r)
}

func (h *handler) setAgent(ctx context.Context, channel Channel, w http.ResponseWriter, r *http.Request) ([]Event, error) {
	payload := &setAgentPayload{}
	err := handlers.DecodeAndValidateJSON(payload, r)
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	// no agent name? ignore this
	if payload.AgentName == "" {
		return nil, handlers.WriteAndLogRequestIgnored(ctx, h, channel, w, r, "Ignoring request, no agent name")
	}

	urn, errURN := urns.NewURNFromParts(channel.Schemes()[0], payload.URN, "", "")
	if errURN != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, errURN)
	}

	contact, err := h.Backend().GetContact(ctx, channel, urn, "", "")
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	_, errSetFieldValue := h.Backend().SetContactCustomField(ctx, contact, "last_agent_name", payload.AgentName)
	if errSetFieldValue != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, errSetFieldValue)
	}

	return nil, WriteDataResponse(ctx, w, http.StatusOK, "Events Handled", []interface{}{})
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

func (h *handler) sendFeedbackRequest(msg Msg, apiURL string, payload *feedbackRequestPayload) (*ChannelLog, error) {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		log := NewChannelLog("unable to build JSON body", msg.Channel(), msg.ID(), "", "", NilStatusCode, "", "", time.Duration(0), err)
		return log, err
	}

	req, _ := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rr, err := utils.MakeHTTPRequest(req)

	// build our channel log
	log := NewChannelLogFromRR("Message Sent", msg.Channel(), msg.ID(), rr).WithError("Message Send Error", err)

	return log, nil
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

	metadata := make(map[string]interface{}, 0)

	if len(msg.QuickReplies()) > 0 {
		buildQuickReplies := make([]string, 0)
		for _, item := range msg.QuickReplies() {
			item = strings.ReplaceAll(item, "\\/", "/")
			item = strings.ReplaceAll(item, "\\\"", "\"")
			item = strings.ReplaceAll(item, "\\\\", "\\")
			buildQuickReplies = append(buildQuickReplies, item)
		}
		metadata["quick_replies"] = buildQuickReplies
	}

	if len(msg.Attachments()) > 0 {
		data.Attachments = msg.Attachments()
	}

	if msg.ReceiveAttachment() != "" {
		metadata["receive_attachment"] = msg.ReceiveAttachment()
	}

	if msg.SharingConfig() != nil {
		metadata["sharing_config"] = msg.SharingConfig()
	}

	data.Metadata = metadata

	// the status that will be written for this message
	status := h.Backend().NewMsgStatusForID(msg.Channel(), msg.ID(), MsgErrored)

	// whether we encountered any errors sending any parts
	hasError := true

	if msg.Topic() == "feedback" {
		feedbackRequestBody := &feedbackRequestBodyPayload{}
		err := json.Unmarshal([]byte(data.Text), feedbackRequestBody)
		if err != nil {
			status.SetStatus(MsgWired)
			return status, err
		}

		feedbackRequest := &feedbackRequestPayload{
			ID:                         data.ID,
			To:                         data.To,
			ToNoPlus:                   data.ToNoPlus,
			From:                       data.From,
			FromNoPlus:                 data.FromNoPlus,
			Channel:                    data.Channel,
			feedbackRequestBodyPayload: *feedbackRequestBody,
		}
		log, err := h.sendFeedbackRequest(msg, address, feedbackRequest)
		status.SetExternalID("")
		status.AddLog(log)
		if err != nil {
			return status, err
		}
		status.SetStatus(MsgWired)
		return status, nil
	}

	// if we have text, send that if we aren't sending it as a caption
	if msg.Text() != "" || len(msg.Attachments()) > 0 {
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

func CreateToken(userURN string, secret string) string {
	var err error
	tokenClaims := jwt.MapClaims{}
	tokenClaims["authorized"] = true
	tokenClaims["userURN"] = userURN
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)
	token, err := at.SignedString([]byte(secret))
	if err != nil {
		return ""
	}
	return token
}

func urnFromToken(tokenString string, secret string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}
	claims, _ := token.Claims.(jwt.MapClaims)
	return claims["userURN"].(string), nil
}

type userPayload struct {
	URN       string                 `json:"urn"`
	Language  string                 `json:"language"`
	UserToken string                 `json:"user_token"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

type msgPayload struct {
	Text          string `json:"text"`
	From          string `json:"from"`
	AttachmentURL string `json:"attachment_url"`
}

type feedbackResponsePayload struct {
	From            string  `json:"from"`
	FeedbackRate    float32 `json:"rate"`
	FeedbackComment string  `json:"comment"`
}

type setAgentPayload struct {
	URN       string `json:"urn"`
	AgentName string `json:"agent_name"`
}

type dataPayload struct {
	ID          string                 `json:"id"`
	Text        string                 `json:"text"`
	To          string                 `json:"to"`
	ToNoPlus    string                 `json:"to_no_plus"`
	From        string                 `json:"from"`
	FromNoPlus  string                 `json:"from_no_plus"`
	Channel     string                 `json:"channel"`
	Metadata    map[string]interface{} `json:"metadata"`
	Attachments []string               `json:"attachments"`
}

type feedbackRequestPayload struct {
	ID         string `json:"id"`
	To         string `json:"to"`
	ToNoPlus   string `json:"to_no_plus"`
	From       string `json:"from"`
	FromNoPlus string `json:"from_no_plus"`
	Channel    string `json:"channel"`

	feedbackRequestBodyPayload
}

type feedbackRequestBodyPayload struct {
	FeedbackRequest feedbackRequestQuestionsPayload `json:"feedback_request"`
}

type feedbackRequestQuestionsPayload struct {
	StarRatingQuestion string `json:"star_rating_question"`
	CommentQuestion    string `json:"comment_question"`
}

type feedbackResponseBodyPayload struct {
	FeedbackResponse feedbackResponseAnswerPayload `json:"feedback_response"`
}

type feedbackResponseAnswerPayload struct {
	StarRatingAnswer float32 `json:"star_rating_answer"`
	CommentAnswer    string  `json:"comment_answer"`
}

type webChatMessagePayload struct {
	Message     string                 `json:"message"`
	Origin      string                 `json:"origin"`
	Metadata    map[string]interface{} `json:"metadata"`
	Attachments []string               `json:"attachments"`
}
