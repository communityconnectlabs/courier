package courier

import (
	"time"

	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/httpx"
	"github.com/nyaruka/gocommon/uuids"
)

// ChannelLogType is the type of channel interaction we are logging
type ChannelLogType string

const (
	ChannelLogTypeUnknown      ChannelLogType = "unknown"
	ChannelLogTypeMsgSend      ChannelLogType = "msg_send"
	ChannelLogTypeMsgStatus    ChannelLogType = "msg_status"
	ChannelLogTypeMsgReceive   ChannelLogType = "msg_receive"
	ChannelLogTypeEventReceive ChannelLogType = "event_receive"
	ChannelLogTypeTokenFetch   ChannelLogType = "token_fetch"
)

type ChannelError struct {
	message string
	code    string
}

func NewChannelError(message, code string) ChannelError {
	return ChannelError{message: message, code: code}
}

func (e *ChannelError) Message() string {
	return e.message
}

func (e *ChannelError) Code() string {
	return e.code
}

// ChannelLog stores the HTTP traces and errors generated by an interaction with a channel.
type ChannelLog struct {
	uuid      uuids.UUID
	type_     ChannelLogType
	channel   Channel
	msgID     MsgID
	recorder  *httpx.Recorder
	httpLogs  []*httpx.Log
	errors    []ChannelError
	createdOn time.Time
	elapsed   time.Duration
}

// NewChannelLogForIncoming creates a new channel log for an incoming request, the type of which won't be known
// until the handler completes.
func NewChannelLogForIncoming(r *httpx.Recorder, ch Channel) *ChannelLog {
	return newChannelLog(ChannelLogTypeUnknown, ch, r, NilMsgID)
}

// NewChannelLogForSend creates a new channel log for a message send
func NewChannelLogForSend(msg Msg) *ChannelLog {
	return newChannelLog(ChannelLogTypeMsgSend, msg.Channel(), nil, msg.ID())
}

// NewChannelLog creates a new channel log with the given type and channel
func NewChannelLog(t ChannelLogType, ch Channel) *ChannelLog {
	return newChannelLog(t, ch, nil, NilMsgID)
}

func newChannelLog(t ChannelLogType, ch Channel, r *httpx.Recorder, mid MsgID) *ChannelLog {
	return &ChannelLog{
		uuid:      uuids.New(),
		type_:     t,
		channel:   ch,
		recorder:  r,
		msgID:     mid,
		createdOn: dates.Now(),
	}
}

// HTTP logs an outgoing HTTP request and response
func (l *ChannelLog) HTTP(t *httpx.Trace) {
	l.httpLogs = append(l.httpLogs, l.traceToLog(t))
}

func (l *ChannelLog) Error(err error) {
	l.errors = append(l.errors, NewChannelError(err.Error(), ""))
}

func (l *ChannelLog) End() {
	if l.recorder != nil {
		// prepend so it's the first HTTP request in the log
		l.httpLogs = append([]*httpx.Log{l.traceToLog(l.recorder.Trace)}, l.httpLogs...)
	}

	l.elapsed = time.Since(l.createdOn)
}

func (l *ChannelLog) UUID() uuids.UUID {
	return l.uuid
}

func (l *ChannelLog) Type() ChannelLogType {
	return l.type_
}

func (l *ChannelLog) SetType(t ChannelLogType) {
	l.type_ = t
}

func (l *ChannelLog) Channel() Channel {
	return l.channel
}

func (l *ChannelLog) MsgID() MsgID {
	return l.msgID
}

func (l *ChannelLog) SetMsgID(id MsgID) {
	l.msgID = id
}

func (l *ChannelLog) HTTPLogs() []*httpx.Log {
	return l.httpLogs
}

func (l *ChannelLog) Errors() []ChannelError {
	return l.errors
}

func (l *ChannelLog) CreatedOn() time.Time {
	return l.createdOn
}

func (l *ChannelLog) Elapsed() time.Duration {
	return l.elapsed
}

func (l *ChannelLog) traceToLog(t *httpx.Trace) *httpx.Log {
	return httpx.NewLog(t, 2048, 50000, nil)
}
