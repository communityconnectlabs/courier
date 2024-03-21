package tembachat

import (
	"testing"

	"github.com/nyaruka/courier"
	. "github.com/nyaruka/courier/handlers"
	"github.com/nyaruka/courier/test"
	"github.com/nyaruka/gocommon/httpx"
)

var incomingCases = []IncomingTestCase{
	{
		Label:                "Message with text",
		URL:                  "/c/twc/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/receive",
		Data:                 `{"chat_id": "65vbbDAQCdPdEWlEhDGy4utO", "secret": "sesame", "events": [{"type": "msg_in", "msg": {"text": "Join"}}]}`,
		ExpectedRespStatus:   200,
		ExpectedBodyContains: "Events Handled",
		ExpectedMsgText:      Sp("Join"),
		ExpectedURN:          "webchat:65vbbDAQCdPdEWlEhDGy4utO",
	},
	{
		Label:                "Message with invalid chat ID",
		URL:                  "/c/twc/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/receive",
		Data:                 `{"chat_id": "xxxxx", "secret": "sesame", "events": [{"type": "msg_in", "msg": {"text": "Join"}}]}`,
		ExpectedRespStatus:   400,
		ExpectedBodyContains: "invalid webchat id: xxxxx",
	},
	{
		Label:                "Chat started event",
		URL:                  "/c/twc/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/receive",
		Data:                 `{"chat_id": "65vbbDAQCdPdEWlEhDGy4utO", "secret": "sesame", "events": [{"type": "chat_started"}]}`,
		ExpectedRespStatus:   200,
		ExpectedBodyContains: "Events Handled",
		ExpectedEvents:       []ExpectedEvent{{Type: courier.EventTypeNewConversation, URN: "webchat:65vbbDAQCdPdEWlEhDGy4utO"}},
	},
	{
		Label:                "Chat started event with invalid chat ID",
		URL:                  "/c/twc/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/receive",
		Data:                 `{"chat_id": "xxxxx", "secret": "sesame", "events": [{"type": "chat_started"}]}`,
		ExpectedRespStatus:   400,
		ExpectedBodyContains: "invalid webchat id: xxxxx",
	},
	{
		Label:                "Msg status update",
		URL:                  "/c/twc/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/receive",
		Data:                 `{"chat_id": "65vbbDAQCdPdEWlEhDGy4utO", "secret": "sesame", "events": [{"type": "msg_status", "status": {"msg_id": 10, "status": "sent"}}]}`,
		ExpectedRespStatus:   200,
		ExpectedBodyContains: "Events Handled",
		ExpectedStatuses:     []ExpectedStatus{{MsgID: 10, Status: courier.MsgStatusSent}},
	},
	{
		Label:                "Missing fields",
		URL:                  "/c/twc/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/receive",
		Data:                 `{"foo": "bar"}`,
		ExpectedRespStatus:   400,
		ExpectedBodyContains: "Field validation for 'ChatID' failed on the 'required' tag",
	},
	{
		Label:                "Incorrect channel secret",
		URL:                  "/c/twc/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/receive",
		Data:                 `{"chat_id": "65vbbDAQCdPdEWlEhDGy4utO", "secret": "xxxxx", "events": [{"type": "msg_in", "msg": {"text": "Join"}}]}`,
		ExpectedRespStatus:   400,
		ExpectedBodyContains: "secret incorrect",
	},
}

func TestIncoming(t *testing.T) {
	chs := []courier.Channel{
		test.NewMockChannel("8eb23e93-5ecb-45ba-b726-3b064e0c56ab", "TWC", "", "", map[string]any{"secret": "sesame"}),
	}

	RunIncomingTestCases(t, chs, newHandler(), incomingCases)
}

var outgoingCases = []OutgoingTestCase{
	{
		Label:   "Flow message",
		MsgText: "Simple message ☺",
		MsgURN:  "webchat:65vbbDAQCdPdEWlEhDGy4utO",
		MockResponses: map[string][]*httpx.MockResponse{
			"http://chatserver:8070/send/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/": {
				httpx.NewMockResponse(200, nil, []byte(`{"status": "queued"}`)),
			},
		},
		ExpectedRequests: []ExpectedRequest{
			{
				Body: `{"chat_id":"65vbbDAQCdPdEWlEhDGy4utO","secret":"sesame","msg":{"id":10,"text":"Simple message ☺","origin":"flow"}}`,
			},
		},
	},
	{
		Label:     "Chat message",
		MsgText:   "Simple message ☺",
		MsgURN:    "webchat:65vbbDAQCdPdEWlEhDGy4utO",
		MsgUserID: 123,
		MockResponses: map[string][]*httpx.MockResponse{
			"http://chatserver:8070/send/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/": {
				httpx.NewMockResponse(200, nil, []byte(`{"status": "queued"}`)),
			},
		},
		ExpectedRequests: []ExpectedRequest{
			{
				Body: `{"chat_id":"65vbbDAQCdPdEWlEhDGy4utO","secret":"sesame","msg":{"id":10,"text":"Simple message ☺","origin":"flow","user_id":123}}`,
			},
		},
	},
	{
		Label:   "400 response",
		MsgText: "Error message",
		MsgURN:  "webchat:65vbbDAQCdPdEWlEhDGy4utO",
		MockResponses: map[string][]*httpx.MockResponse{
			"http://chatserver:8070/send/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/": {
				httpx.NewMockResponse(400, nil, []byte(`{"error": "invalid"}`)),
			},
		},
		ExpectedRequests: []ExpectedRequest{
			{
				Body: `{"chat_id":"65vbbDAQCdPdEWlEhDGy4utO","secret":"sesame","msg":{"id":10,"text":"Error message","origin":"flow"}}`,
			},
		},
		ExpectedError: courier.ErrResponseUnexpected,
	},
	{
		Label:   "500 response",
		MsgText: "Error message",
		MsgURN:  "webchat:65vbbDAQCdPdEWlEhDGy4utO",
		MockResponses: map[string][]*httpx.MockResponse{
			"http://chatserver:8070/send/8eb23e93-5ecb-45ba-b726-3b064e0c56ab/": {
				httpx.NewMockResponse(500, nil, []byte(`Gateway Error`)),
			},
		},
		ExpectedRequests: []ExpectedRequest{
			{
				Body: `{"chat_id":"65vbbDAQCdPdEWlEhDGy4utO","secret":"sesame","msg":{"id":10,"text":"Error message","origin":"flow"}}`,
			},
		},
		ExpectedError: courier.ErrConnectionFailed,
	},
}

func TestOutgoing(t *testing.T) {
	ch := test.NewMockChannel("8eb23e93-5ecb-45ba-b726-3b064e0c56ab", "TWC", "", "", map[string]any{"secret": "sesame"})

	RunOutgoingTestCases(t, ch, newHandler(), outgoingCases, []string{"sesame"}, nil)
}
