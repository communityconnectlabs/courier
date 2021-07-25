package ccl_test_channel

import (
	"github.com/nyaruka/courier"
	. "github.com/nyaruka/courier/handlers"
	"net/http/httptest"
	"testing"
)

var testChannels = []courier.Channel{
	courier.NewMockChannel("8eb23e93-5ecb-45ba-b726-3b064e0c56ab", "CCL", "2020", "US", nil),
}

var (
	receiveURL = "/c/ccl/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/receive/"
	statusURL  = "/c/ccl/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/status/"

	emptyReceive = "empty"
	validReceive = "text=Msg&to=21512&id=ec9adc86-51d5-4bc8-8eb0-d8ab0bb53dc3&from=%2B254791541111"
	invalidURN   = "text=Msg&to=21512&id=ec9adc86-51d5-4bc8-8eb0-d8ab0bb53dc3&from=MTN"
	missingText  = "to=21512&id=ec9adc86-51d5-4bc8-8eb0-d8ab0bb53dc3&from=%2B254791541111"

	missingStatus = "id=ATXid_dda018a640edfcc5d2ce455de3e4a6e7"
	invalidStatus = "id=ATXid_dda018a640edfcc5d2ce455de3e4a6e7&status=Borked"
	validStatus   = "id=ATXid_dda018a640edfcc5d2ce455de3e4a6e7&status=Success"
)

var testCases = []ChannelHandleTestCase{
	{Label: "Receive Valid", URL: receiveURL, Data: validReceive, Status: 200, Response: "Message Accepted",
		Text: Sp("Msg"), URN: Sp("tel:+254791541111"), ExternalID: Sp("ec9adc86-51d5-4bc8-8eb0-d8ab0bb53dc3")},
	{Label: "Receive Empty", URL: receiveURL, Data: emptyReceive, Status: 400, Response: "field 'id' required"},
	{Label: "Receive Missing Text", URL: receiveURL, Data: missingText, Status: 400, Response: "field 'text' required"},
	{Label: "Invalid URN", URL: receiveURL, Data: invalidURN, Status: 400, Response: "phone number supplied is not a number"},
	{Label: "Status Invalid", URL: statusURL, Status: 400, Data: invalidStatus, Response: "unknown status"},
	{Label: "Status Missing", URL: statusURL, Status: 400, Data: missingStatus, Response: "field 'status' required"},
	{Label: "Status Valid", URL: statusURL, Status: 200, Data: validStatus, Response: `"status":"D"`},
}

func TestHandler(t *testing.T) {
	RunChannelTestCases(t, testChannels, newHandler(), testCases)
}

func BenchmarkHandler(b *testing.B) {
	RunChannelBenchmarks(b, testChannels, newHandler(), testCases)
}

// setSendURL takes care of setting the sendURL to call
func setSendURL(s *httptest.Server, h courier.ChannelHandler, c courier.Channel, m courier.Msg) {
	sendURL = s.URL
}

var defaultSendTestCases = []ChannelSendTestCase{
	{Label: "Plain Send",
		Text: "Simple Message ☺", URN: "tel:+250788383383",
		Status: "W", ExternalID: "1002",
		ResponseBody: `{ "SMSMessageData": {"Recipients": [{"status": "Success", "messageId": "1002"}] } }`, ResponseStatus: 200,
		Headers:    map[string]string{"apikey": "KEY"},
		PostParams: map[string]string{"message": "Simple Message ☺", "username": "Username", "to": "+250788383383", "from": "2020"},
		SendPrep:   setSendURL},
	{Label: "Send Attachment",
		Text: "My pic!", URN: "tel:+250788383383", Attachments: []string{"image/jpeg:https://foo.bar/image.jpg"},
		Status: "W", ExternalID: "1002",
		ResponseBody: `{ "SMSMessageData": {"Recipients": [{"status": "Success", "messageId": "1002"}] } }`, ResponseStatus: 200,
		PostParams: map[string]string{"message": "My pic!\nhttps://foo.bar/image.jpg"},
		SendPrep:   setSendURL},
	{Label: "No External Id",
		Text: "No External ID", URN: "tel:+250788383383",
		Status:       "E",
		ResponseBody: `{ "SMSMessageData": {"Recipients": [{"status": "Failed" }] } }`, ResponseStatus: 200,
		PostParams: map[string]string{"message": `No External ID`},
		SendPrep:   setSendURL},
	{Label: "Error Sending",
		Text: "Error Message", URN: "tel:+250788383383",
		Status:       "E",
		ResponseBody: `{ "error": "failed" }`, ResponseStatus: 401,
		PostParams: map[string]string{"message": `Error Message`},
		SendPrep:   setSendURL},
}

func TestSending(t *testing.T) {
	var defaultChannel = courier.NewMockChannel("8eb23e93-5ecb-45ba-b726-3b064e0c56ab", "CCL", "2020", "US",
		map[string]interface{}{
			courier.ConfigUsername: "Username",
			courier.ConfigAPIKey:   "KEY",
		})

	RunChannelSendTestCases(t, defaultChannel, newHandler(), defaultSendTestCases, nil)
}
