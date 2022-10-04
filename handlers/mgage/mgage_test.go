package mgage

import (
	"github.com/nyaruka/courier"
	. "github.com/nyaruka/courier/handlers"
	"github.com/nyaruka/gocommon/urns"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

var testChannels = []courier.Channel{
	courier.NewMockChannel("58b70770-b76c-40e6-8755-8abb65611839", "MGA", "00338683", "US", map[string]interface{}{}),
}

var helloMsg = `{
	"sender": "+18889099091",
	"receiver": "00338683",
	"encoding": "GSM7",
	"text": "Hello, world!"
}`

var testCases = []ChannelHandleTestCase{
	//{
	//	Label:    "Receive Valid Message",
	//	URL:      "/c/mga/receive",
	//	Data:     helloMsg,
	//	Status:   200,
	//	Response: "Accepted",
	//	Text:     Sp("Hello, world!"),
	//	URN:      Sp("tel:+18889099091"),
	//	Date:     nil,
	//},
	//{
	//	Label:    "Receive Enroute Status",
	//	URL:      "/c/mga/status",
	//	Data:     `{
	//	  "msg_ref": "1234567890ABCDEF",
	//	  "status": "U",
	//	  "data": {
	//		"CarrierID": "ABCDEF1234567890"
    //      }
	//	}`,
	//	Status:   200,
	//	Response: "Accepted",
	//	NoQueueErrorCheck: true,
	//	NoInvalidChannelCheck: true,
	//},
	//{
	//	Label:    "Receive Delivered Status",
	//	URL:      "/c/mga/status",
	//	Data:     `{
	//	  "msg_ref": "ABCDEF1234567890",
	//	  "status": "D",
	//	  "data": {}
	//	}`,
	//	Status:   200,
	//	Response: "Accepted",
	//	NoQueueErrorCheck: true,
	//	NoInvalidChannelCheck: true,
	//},
	{
		Label:    "Receive Delivered Status",
		URL:      "/c/mga/status",
		Data:     `{
		  "msg_ref": "ABCDEF1234567890",
		  "status": "D",
		  "data": {}
		}`,
		Status:   200,
		Response: "Accepted",
		NoQueueErrorCheck: true,
		NoInvalidChannelCheck: true,
	},
}

var defaultSendTestCases = []ChannelSendTestCase{
	{
		Label:          "Plain Send",
		Text:           "Hello, world!",
		URN:            "tel:+18889099091",
		Status:         "W",
		ResponseBody:   ``,
		ResponseStatus: 200,
	},
	{
		Label:          "Multiple Segments",
		Text:           "Tarty giant letter generator uses text symbols ▓▒░▄█▀▌▐─. █▀█ █▄█ ▀█▀ / █▬█ █ ▀█▀ font (no, not ▟▛ █▬█ █ ▜▛ font) uses a different bunch of symbols to fonm letters. By the way, check out my collection of text drawings. ≧^◡^≦ When you'll find a copy of my big text generators around the internet - there's plenty of copies, please know that this is the actual original and we actually designed all of these big letters with my friends, including ASCII Text Art Generator and 𝗧𝗲𝘅𝘁 font copy paste. Proud and angry!",
		URN:            "tel:+18889099091",
		Status:         "W",
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

func TestHandler(t *testing.T) {
	smppService := buildMockSMPPService(testCases)
	defer smppService.Close()

	var defaultChannel = courier.NewMockChannel(
		"58b70770-b76c-40e6-8755-8abb65611839",
		"MGA",
		"+12211414154",
		"US",
		map[string]interface{}{},
	)
	defaultChannel.SetScheme(urns.ExternalScheme)

	RunChannelTestCases(t, testChannels, newHandler(), testCases)
}

func setupBackendAndServer(smppTestServerUrl string) func() (*courier.MockBackend, courier.Server) {
	return func() (mb *courier.MockBackend, s courier.Server) {
		mb = courier.NewMockBackend()

		logger := logrus.New()
		logger.Out = ioutil.Discard
		logrus.SetOutput(ioutil.Discard)

		config := courier.NewConfig()
		config.SMPPServerEndpoint = smppTestServerUrl

		s = courier.NewServerWithLogger(config, mb, logger)
		return
	}
}

func TestSending(t *testing.T) {
	smppService := buildMockSMPPService(testCases)
	defer smppService.Close()

	var defaultChannel = courier.NewMockChannel(
		"58b70770-b76c-40e6-8755-8abb65611839",
		"MGA",
		"+12211414154",
		"US",
		map[string]interface{}{},
	)
	defaultChannel.SetScheme(urns.TelScheme)

	RunChannelSendTestCases(t, defaultChannel, newHandler(), defaultSendTestCases, setupBackendAndServer(smppService.URL))
}
