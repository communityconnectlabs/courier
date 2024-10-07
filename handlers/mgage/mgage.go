package mgage

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/nyaruka/gocommon/uuids"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nyaruka/courier"
	"github.com/nyaruka/courier/handlers"
	"github.com/nyaruka/courier/utils"
	"github.com/nyaruka/gocommon/gsm7"
	"github.com/nyaruka/gocommon/urns"
	"github.com/nyaruka/phonenumbers"
	"github.com/pkg/errors"
)

func init() {
	courier.RegisterHandler(newHandler())
}

type handler struct {
	handlers.BaseHandler
}

func newHandler() courier.ChannelHandler {
	return &handler{handlers.NewBaseHandlerWithParams("MGA", "mGage", false)}
}

// Initialize is called by the engine once everything is loaded
func (h *handler) Initialize(s courier.Server) error {
	h.SetServer(s)
	s.AddHandlerRoute(h, http.MethodPost, "receive", h.receiveMessage)
	s.AddHandlerRoute(h, http.MethodPost, "status", h.receiveStatus)
	s.AddHandlerRoute(h, http.MethodPost, "mms_receive", h.receiveMMSMessage)
	s.AddHandlerRoute(h, http.MethodPost, "mms_status", h.receiveMMSStatus)
	return nil
}

func (h *handler) SendLongcodeMsgMMS(msg courier.Msg) (*utils.RequestResponse, error) {
	sendURL := h.Server().Config().KaleyraMMSLongcodeEndpoint
	username := h.Server().Config().KaleyraMMSUsername
	password := h.Server().Config().KaleyraMMSPassword

	channelAddress := strings.TrimLeft(msg.Channel().Address(), "+")
	destination := strings.TrimLeft(msg.URN().Path(), "+")

	attachment := msg.Attachments()[0]
	parts := strings.SplitN(attachment, ":", 2)
	mimeType := parts[0]
	fullURL := parts[1]

	mimeParts := strings.SplitN(mimeType, "/", 2)
	mediaType := mimeParts[0]

	// Extract filename from URL
	urlParts := strings.Split(fullURL, "/")
	filename := urlParts[len(urlParts)-1]
	maxLength := 63
	if utf8.RuneCountInString(filename) > maxLength {
		filename = filename[:maxLength]
	}

	clientTranscationId := fmt.Sprintf("%s-%s", msg.ID().String(), string(uuids.New()))
	if utf8.RuneCountInString(clientTranscationId) > maxLength {
		clientTranscationId = clientTranscationId[:maxLength]
	}

	mmsMsgData := make([]mmsLongcodeMessage, 0)
	mmsMsg := mmsLongcodeMessage{
		DisplayName: filename,
		MediaType:   mediaType,
		MediaUrl:    fullURL,
	}
	mmsMsgData = append(mmsMsgData, mmsMsg)

	payload := &mmsLongcodePayload{
		RequestId:       clientTranscationId,
		From:            channelAddress,
		To:              []string{destination},
		RefMessageId:    msg.ID().String(),
		AllowAdaptation: "true",
		ForwardLock:     "false",
		Message:         mmsMsgData,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest(http.MethodPost, sendURL, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(username, password)

	rr, err := utils.MakeHTTPRequest(req)

	return rr, err
}

func (h *handler) SendShortcodeMsgMMS(msg courier.Msg) (*utils.RequestResponse, error) {
	sendURL := h.Server().Config().KaleyraMMSEndpoint
	username := h.Server().Config().KaleyraMMSUsername
	password := h.Server().Config().KaleyraMMSPassword

	channelAddress := strings.TrimLeft(msg.Channel().Address(), "+")
	productCode := fmt.Sprintf("CCX_%s", channelAddress)

	attachment := msg.Attachments()[0]
	parts := strings.SplitN(attachment, ":", 2)
	mimeType := parts[0]
	fullURL := parts[1]

	// Extract filename from URL
	urlParts := strings.Split(fullURL, "/")
	filename := urlParts[len(urlParts)-1]
	maxLength := 63
	if utf8.RuneCountInString(filename) > maxLength {
		filename = filename[:maxLength]
	}

	destination := fmt.Sprintf("tel:%s", strings.TrimLeft(msg.URN().Path(), "+"))
	clientTranscation := fmt.Sprintf("%s-%s", msg.ID().String(), string(uuids.New()))
	if utf8.RuneCountInString(clientTranscation) > maxLength {
		clientTranscation = clientTranscation[:maxLength]
	}

	form := url.Values{
		"serviceCode":         []string{channelAddress},
		"destination":         []string{destination},
		"subject":             []string{""},
		"isSubjectEncoded":    []string{"true"},
		"senderID":            []string{channelAddress},
		"clientTransactionID": []string{clientTranscation},
		"contentFileName":     []string{filename},
		"contentURL":          []string{fullURL},
		"contentType":         []string{mimeType},
		"contentFLock":        []string{"false"},
		"productCode":         []string{productCode},
		"action":              []string{"CONTENT"},
		"allowAdaptation":     []string{"true"},
		"priority":            []string{"normal"},
	}

	req, err := http.NewRequest(http.MethodPost, sendURL+"?"+form.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(username, password)

	rr, err := utils.MakeHTTPRequest(req)
	return rr, err
}

func (h *handler) SendMsg(ctx context.Context, msg courier.Msg) (courier.MsgStatus, error) {
	if msg.URN().Scheme() != urns.TelScheme {
		return nil, fmt.Errorf("wrong urn scheme for the current mGage channel type")
	}

	msgEncoding := GSM7
	isGSM := gsm7.IsValid(msg.Text())

	charsToCheck := []string{"@", "ñ", "é", "ü", "€", "_"}

	foundUCS := false
	for _, char := range charsToCheck {
		if strings.Contains(msg.Text(), char) {
			foundUCS = true
			break
		}
	}

	if !isGSM || foundUCS {
		msgEncoding = UCS2
	}

	status := h.Backend().NewMsgStatusForID(msg.Channel(), msg.ID(), courier.MsgWired)

	if len(msg.Attachments()) > 0 {
		channelAddress := msg.Channel().Address()
		var rr *utils.RequestResponse
		var err error

		if len(channelAddress) > 6 {
			// using REST API for Longcode MMS
			rr, err = h.SendLongcodeMsgMMS(msg)
		} else {
			// using REST API for Shortcode MMS
			rr, err = h.SendShortcodeMsgMMS(msg)
		}

		log := courier.NewChannelLogFromRR("Message Sent", msg.Channel(), msg.ID(), rr).WithError("Message Send Error", err)
		if err != nil {
			status.SetStatus(courier.MsgFailed)
		}

		status.AddLog(log)
	}

	if h.shouldSplit(msg.Text(), msgEncoding) {
		parts := h.encodeSplit(msg.Text(), msgEncoding)
		partsLength := len(parts)
		for index, part := range parts {
			msgID, _ := strconv.Atoi(msg.ID().String())
			payload := &moPayload{
				ID:       int32(msgID),
				Sender:   msg.Channel().Address(),
				Receiver: msg.URN().Path(),
				Text:     part,
				Encoding: string(msgEncoding),
				PartNum:  int32(index + 1),
				Parts:    int32(partsLength),
			}

			rr, err := h.sendToSMPP(payload)
			log := courier.NewChannelLogFromRR("Message Sent", msg.Channel(), msg.ID(), rr).WithError("Message Send Error", err)

			if err != nil {
				status.SetStatus(courier.MsgFailed)
				_ = h.Backend().WriteSMPPLog(ctx, &courier.SMPPLog{
					ChannelID: msg.Channel().ID(),
					MsgID:     msg.ID(),
					Status:    courier.MsgFailed,
					CreatedOn: time.Now(),
				})
			}

			status.AddLog(log)
		}
	} else {
		msgID, _ := strconv.Atoi(msg.ID().String())
		payload := &moPayload{
			ID:       int32(msgID),
			Sender:   msg.Channel().Address(),
			Receiver: msg.URN().Path(),
			Text:     msg.Text(),
			Encoding: string(msgEncoding),
			PartNum:  1,
			Parts:    1,
		}

		rr, err := h.sendToSMPP(payload)
		log := courier.NewChannelLogFromRR("Message Sent", msg.Channel(), msg.ID(), rr).WithError("Message Send Error", err)

		if err != nil {
			status.SetStatus(courier.MsgFailed)
			_ = h.Backend().WriteSMPPLog(ctx, &courier.SMPPLog{
				ChannelID: msg.Channel().ID(),
				MsgID:     msg.ID(),
				Status:    courier.MsgFailed,
				CreatedOn: time.Now(),
			})
		}

		status.AddLog(log)
	}

	_ = h.Backend().WriteSMPPLog(ctx, &courier.SMPPLog{
		ChannelID: msg.Channel().ID(),
		MsgID:     msg.ID(),
		Status:    courier.MsgWired,
		CreatedOn: time.Now(),
	})

	return status, nil
}

// GetChannel returns the channel
func (h *handler) GetChannel(ctx context.Context, r *http.Request) (courier.Channel, error) {
	if r.Method == http.MethodGet {
		return nil, nil
	}

	payload := &channelPayload{}
	err := handlers.DecodeAndValidateJSON(payload, r)
	if err != nil {
		return nil, err
	}

	if channelAddress := payload.Receiver; channelAddress != "" {
		if isLongCode := len(channelAddress) >= 10; isLongCode {
			parsed, err := phonenumbers.Parse(channelAddress, "US")
			if err != nil {
				return nil, err
			}
			channelAddress = phonenumbers.Format(parsed, phonenumbers.E164)
		}
		return h.Backend().GetChannelByAddress(ctx, "MGA", courier.ChannelAddress(channelAddress))
	} else if payload.MsgID != 0 {
		return h.Backend().GetMsgChannel(ctx, "MGA", courier.MsgID(payload.MsgID), "")
	} else if payload.MsgRef != "" {
		channel, err := h.Backend().GetMsgChannel(ctx, "MGA", courier.MsgID(0), payload.MsgRef)
		// allow empty channel in case the Gateway ID wasn't received yet
		if errors.Is(err, courier.ErrChannelNotFound) {
			return &EmptyMGAChannel{}, nil
		}
		return channel, nil
	}
	return nil, errors.New("At least one of [MsgID, MsgRef, Receiver] must be provided.")
}

func (h *handler) receiveMessage(ctx context.Context, channel courier.Channel, w http.ResponseWriter, r *http.Request) ([]courier.Event, error) {
	payload := &moPayload{}
	err := handlers.DecodeAndValidateJSON(payload, r)
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	// create our date from the timestamp
	date := time.Now().UTC()

	// format contact phone number
	senderPhoneNumber := payload.Sender
	if isLongCode := len(senderPhoneNumber) >= 10; isLongCode {
		parsed, err := phonenumbers.Parse(senderPhoneNumber, "US")
		if err != nil {
			return nil, err
		}
		senderPhoneNumber = phonenumbers.Format(parsed, phonenumbers.E164)
	}

	// create our URN
	urn, err := urns.NewURNFromParts(urns.TelScheme, senderPhoneNumber, "", "")
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	// build our msg
	msg := h.Backend().NewIncomingMsg(channel, urn, payload.Text).WithReceivedOn(date)

	// check whether the organization has opt-out enabled or not
	optOutDisabled := channel.OrgConfigForKey(utils.OptOutDisabled, false)
	if optOutDisabled == false {
		// check whether the text contains opt-out words
		if utils.CheckOptOutKeywordPresence(msg.Text()) {
			textMessage := channel.OrgConfigForKey(utils.OptOutMessageBackKey, utils.OptOutDefaultMessageBack)
			msgBack := h.Backend().NewOutgoingMsg(
				channel, courier.MsgID(0), urn, textMessage.(string), true, []string{}, "", "",
			)
			_, err := h.SendMsg(ctx, msgBack)
			if err != nil {
				return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
			}
		}
	}

	return handlers.WriteMsgsAndResponse(ctx, h, []courier.Msg{msg}, w, r)
}

func (h *handler) receiveStatus(ctx context.Context, channel courier.Channel, w http.ResponseWriter, r *http.Request) ([]courier.Event, error) {
	var status courier.MsgStatus

	payload := &eventPayload{}
	err := handlers.DecodeAndValidateJSON(payload, r)
	if err != nil {
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, err)
	}

	switch payload.Status {
	case courier.MsgSent:
		if payload.MsgID == 0 {
			return nil, handlers.WriteAndLogRequestIgnored(ctx, h, channel, w, r, "no msg status, ignoring")
		}
		status = h.Backend().NewMsgStatusForID(channel, courier.MsgID(payload.MsgID), courier.MsgSent)
		status.SetExternalID(payload.MsgRef)
		status.SetGatewayID(payload.MsgRef)
		return handlers.WriteMsgStatusAndResponse(ctx, h, channel, status, w, r)
	case courier.MsgEnroute:
		if payload.MsgRef == "" {
			return nil, handlers.WriteAndLogRequestIgnored(ctx, h, channel, w, r, "no msg status, ignoring")
		}
		status = h.Backend().NewMsgStatusForExternalID(channel, payload.MsgRef, courier.MsgEnroute)
		dm, ok := payload.Data.(map[string]interface{})
		if !ok {
			return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, errors.New("no data to update status"))
		}

		carrierID, ok := dm["CarrierID"]
		if !ok {
			return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, errors.New("no carrier id to update status"))
		}
		status.SetGatewayID(payload.MsgRef)
		status.SetCarrierID(carrierID.(string))

		// store channel logs if exact channel can't be defined at the moment
		if _, isEmptyMGAChannel := channel.(*EmptyMGAChannel); isEmptyMGAChannel {
			processStatusLogs(status, channel, r)
		}
		return handlers.WriteMsgStatusAndResponse(ctx, h, channel, status, w, r)
	case courier.MsgDelivered:
		if payload.MsgRef == "" {
			return nil, handlers.WriteAndLogRequestIgnored(ctx, h, channel, w, r, "no msg status, ignoring")
		}
		msgIDMap, err := h.Backend().GetMsgIDByExternalID(ctx, payload.MsgRef)
		if err == sql.ErrNoRows || (msgIDMap != nil && msgIDMap.ID() == courier.NilMsgID) {
			// save channel logs if exact channel can't be defined at the moment
			status = h.Backend().NewMsgStatusForExternalID(channel, payload.MsgRef, courier.MsgDelivered)
			status.SetCarrierID(payload.MsgRef)
			processStatusLogs(status, channel, r)
			return handlers.WriteMsgStatusAndResponse(ctx, h, channel, status, w, r)
		} else if err == nil {
			// save normal channel log
			status = h.Backend().NewMsgStatusForID(channel, msgIDMap.ID(), courier.MsgDelivered)
			return handlers.WriteMsgStatusAndResponse(ctx, h, channel, status, w, r)
		}
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, errors.Wrap(err, "failed to get message data"))
	case courier.MsgErrored:
		if payload.MsgRef == "" {
			return nil, handlers.WriteAndLogRequestIgnored(ctx, h, channel, w, r, "no msg status, ignoring")
		}
		msgIDMap, err := h.Backend().GetMsgIDByExternalID(ctx, payload.MsgRef)
		if err == sql.ErrNoRows || (msgIDMap != nil && msgIDMap.ID() == courier.NilMsgID) {
			// save channel logs if exact channel can't be defined at the moment
			status = h.Backend().NewMsgStatusForExternalID(channel, payload.MsgRef, courier.MsgErrored)
			status.SetGatewayID(payload.MsgRef)
			processStatusLogs(status, channel, r)
			return handlers.WriteMsgStatusAndResponse(ctx, h, channel, status, w, r)
		} else if err == nil {
			// save normal channel log
			status = h.Backend().NewMsgStatusForID(channel, msgIDMap.ID(), courier.MsgErrored)
			return handlers.WriteMsgStatusAndResponse(ctx, h, channel, status, w, r)
		}
		return nil, handlers.WriteAndLogRequestError(ctx, h, channel, w, r, errors.Wrap(err, "failed to get message data"))
	case courier.MsgFailed:
		if payload.MsgID == 0 {
			return nil, handlers.WriteAndLogRequestIgnored(ctx, h, channel, w, r, "no msg status, ignoring")
		}
		status = h.Backend().NewMsgStatusForID(channel, courier.MsgID(payload.MsgID), courier.MsgFailed)
		return handlers.WriteMsgStatusAndResponse(ctx, h, channel, status, w, r)
	}
	return nil, nil
}

func processStatusLogs(status courier.MsgStatus, channel courier.Channel, r *http.Request) {
	if _, isEmptyMGA := channel.(*EmptyMGAChannel); isEmptyMGA {
		var duration time.Duration = -1

		// Trim out cookie header, should never be part of authentication and can leak auth to channel logs
		r.Header.Del("Cookie")
		request, err := httputil.DumpRequest(r, true)
		if err != nil {
			// skip creation of log if any error
			return
		}
		url := fmt.Sprintf("https://%s%s", r.Host, r.URL.RequestURI())

		// Prepare response data
		response := map[string]interface{}{
			"message": "Status Update Accepted",
			"data":    []interface{}{courier.NewStatusData(status)},
		}
		responseJson, _ := json.Marshal(response)
		status.AddLog(courier.NewChannelLog("Status Updated", channel, courier.NilMsgID, r.Method, url, http.StatusOK, string(request), string(responseJson), duration, err))
	}
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

func (h *handler) receiveMMSMessage(ctx context.Context, channel courier.Channel, w http.ResponseWriter, r *http.Request) ([]courier.Event, error) {
	return nil, nil
}

func (h *handler) receiveMMSStatus(ctx context.Context, channel courier.Channel, w http.ResponseWriter, r *http.Request) ([]courier.Event, error) {
	return nil, nil
}

type MsgEncoding string

const (
	GSM7     MsgEncoding = "GSM7"
	UCS2     MsgEncoding = "UCS2"
	SmMsgLen uint        = 140
)

type moPayload struct {
	ID       int32  `json:"id,omitempty"`
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Encoding string `json:"encoding"`
	Text     string `json:"text"`
	Parts    int32  `json:"parts"`
	PartNum  int32  `json:"part_num"`
}

type eventPayload struct {
	MsgID  int32                  `json:"msg_id"`
	MsgRef string                 `json:"msg_ref"`
	Status courier.MsgStatusValue `json:"status"`
	Data   interface{}            `json:"data"`
}

type channelPayload struct {
	MsgID    int32  `json:"msg_id,omitempty"`
	MsgRef   string `json:"msg_ref,omitempty"`
	Receiver string `json:"receiver,omitempty"`
}

type mmsLongcodePayload struct {
	CustomerId      string               `json:"customerId,omitempty"`
	RequestId       string               `json:"requestId"`
	From            string               `json:"from"`
	To              []string             `json:"to"`
	Subject         string               `json:"subject,omitempty"`
	RefMessageId    string               `json:"refMessageId,omitempty"`
	ReportingKey1   string               `json:"reportingKey1,omitempty"`
	ReportingKey2   string               `json:"reportingKey2,omitempty"`
	AllowAdaptation string               `json:"allowAdaptation"`
	ForwardLock     string               `json:"forwardLock"`
	Message         []mmsLongcodeMessage `json:"message"`
}

type mmsLongcodeMessage struct {
	Text        string `json:"text,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	MediaType   string `json:"mediaType,omitempty"`
	MediaUrl    string `json:"mediaUrl,omitempty"`
}
