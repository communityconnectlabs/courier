package smpp

import (
	"context"
	"github.com/nyaruka/courier"
	. "github.com/nyaruka/courier/handlers"
	"net/http"
	"net/http/httptest"
	"testing"
)

var testChannels = []courier.Channel{
	courier.NewMockChannel("58b70770-b76c-40e6-8755-8abb65611839", "SMP", "00338683", "US", map[string]interface{}{
		"sms_center":          "smscsim.melroselabs.com:2775",
		"system_id":           "111111",
		"password":            "111111",
		"phone_number":        "00111111",
		"allow_international": false,
	}),
}

var helloMsg = `{
  	"channel": "58b70770-b76c-40e6-8755-8abb65611839",
  	"contact": "99338683",
  	"text": "Hello World"
}`

var testCases = []ChannelHandleTestCase{
	{
		Label:    "Receive Valid Message",
		URL:      "/c/smp/58b70770-b76c-40e6-8755-8abb65611839/receive/",
		Data:     helloMsg,
		Status:   200,
		Response: "Accepted",
		Text:     Sp("Hello World"),
		URN:      Sp("tel:99338683"),
		Date:     nil,
	},
}

var defaultSendTestCases = []ChannelSendTestCase{
	{
		Label:          "Plain Send",
		Text:           "Hello World",
		URN:            "tel:99338683",
		Status:         "S",
		ResponseBody:   ``,
		ResponseStatus: 200,
	},
}

func buildMockSMPPService(testCases []ChannelHandleTestCase) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()
	}))

	return server
}

// defining decorator struct for handler that will mock SMPP server URL
type smppTestHandler struct {
	*handler
	SMPPServerEndpoint string
}

func (h *smppTestHandler) Initialize(s courier.Server) error {
	if h.SMPPServerEndpoint != "" {
		s.Config().SMPPServerEndpoint = h.SMPPServerEndpoint
	}
	return h.handler.Initialize(s)
}
func (h *smppTestHandler) SendMsg(ctx context.Context, msg courier.Msg) (courier.MsgStatus, error) {
	return h.handler.SendMsg(ctx, msg)
}
func (h *smppTestHandler) receiveMessage(ctx context.Context, channel courier.Channel, w http.ResponseWriter, r *http.Request) ([]courier.Event, error) {
	return h.handler.receiveMessage(ctx, channel, w, r)
}

// func that will return decorated handler
func newTestSMPPHandler(mockedEndpoint string) courier.ChannelHandler {
	return &smppTestHandler{
		newHandler().(*handler),
		mockedEndpoint,
	}
}

func TestHandler(t *testing.T) {
	smppService := buildMockSMPPService(testCases)
	defer smppService.Close()

	RunChannelTestCases(t, testChannels, newTestSMPPHandler(smppService.URL), testCases)
}

func TestSending(t *testing.T) {
	smppService := buildMockSMPPService(testCases)
	defer smppService.Close()

	var defaultChannel = courier.NewMockChannel(
		"58b70770-b76c-40e6-8755-8abb65611839",
		"SMP",
		"00338683",
		"US",
		map[string]interface{}{
			"sms_center":          "smscsim.melroselabs.com:2775",
			"system_id":           "111111",
			"password":            "111111",
			"phone_number":        "00111111",
			"allow_international": false,
		},
	)

	RunChannelSendTestCases(t, defaultChannel, newTestSMPPHandler(smppService.URL), defaultSendTestCases, nil)
}
