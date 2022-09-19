package rapidpro

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nyaruka/courier"
	"github.com/nyaruka/gocommon/jsonx"
)

const insertLogSQL = `
INSERT INTO channels_channellog( uuid,  log_type,  channel_id,  msg_id,  http_logs,  errors,  is_error,  created_on,  elapsed_ms)
                         VALUES(:uuid, :log_type, :channel_id, :msg_id, :http_logs, :errors, :is_error, :created_on, :elapsed_ms)`

// ChannelLog is our DB specific struct for logs
type ChannelLog struct {
	UUID      courier.ChannelLogUUID `db:"uuid"`
	Type      courier.ChannelLogType `db:"log_type"`
	ChannelID courier.ChannelID      `db:"channel_id"`
	MsgID     courier.MsgID          `db:"msg_id"`
	HTTPLogs  json.RawMessage        `db:"http_logs"`
	Errors    json.RawMessage        `db:"errors"`
	IsError   bool                   `db:"is_error"`
	CreatedOn time.Time              `db:"created_on"`
	ElapsedMS int                    `db:"elapsed_ms"`
}

// RowID satisfies our batch.Value interface, we are always inserting logs so we have no row id
func (l *ChannelLog) RowID() string {
	return ""
}

type channelError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

// queues the passed in channel log the committer, we do not queue on errors but instead just throw away the log
func queueChannelLog(ctx context.Context, b *backend, clog *courier.ChannelLog) error {
	dbChan := clog.Channel().(*DBChannel)

	// if we have an error or a non 2XX/3XX http response then this log is marked as an error
	isError := len(clog.Errors()) > 0
	if !isError {
		for _, l := range clog.HTTPLogs() {
			if l.StatusCode < 200 || l.StatusCode >= 400 {
				isError = true
				break
			}
		}
	}

	errors := make([]channelError, len(clog.Errors()))
	for i, e := range clog.Errors() {
		errors[i] = channelError{Message: e.Message(), Code: e.Code()}
	}

	// create our value for committing
	v := &ChannelLog{
		UUID:      clog.UUID(),
		Type:      clog.Type(),
		ChannelID: dbChan.ID(),
		MsgID:     clog.MsgID(),
		HTTPLogs:  jsonx.MustMarshal(clog.HTTPLogs()),
		Errors:    jsonx.MustMarshal(errors),
		IsError:   isError,
		CreatedOn: clog.CreatedOn(),
		ElapsedMS: int(clog.Elapsed() / time.Millisecond),
	}

	// queue it
	b.logCommitter.Queue(v)
	return nil
}
