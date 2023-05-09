package courier

import "github.com/nyaruka/gocommon/urns"

// MsgStatusValue is the status of a message
type MsgStatusValue string

// Possible values for MsgStatus
const (
	MsgPending   MsgStatusValue = "P"
	MsgQueued    MsgStatusValue = "Q"
	MsgSent      MsgStatusValue = "S"
	MsgWired     MsgStatusValue = "W"
	MsgEnroute   MsgStatusValue = "U"
	MsgErrored   MsgStatusValue = "E"
	MsgHandled   MsgStatusValue = "H"
	MsgDelivered MsgStatusValue = "D"
	MsgFailed    MsgStatusValue = "F"
	NilMsgStatus MsgStatusValue = ""
)

//-----------------------------------------------------------------------------
// MsgStatusUpdate Interface
//-----------------------------------------------------------------------------

// MsgStatus represents a status update on a message
type MsgStatus interface {
	EventID() int64

	ChannelID() ChannelID
	ChannelUUID() ChannelUUID
	ID() MsgID

	SetUpdatedURN(old, new urns.URN) error
	UpdatedURN() (old, new urns.URN)
	HasUpdatedURN() bool

	ExternalID() string
	SetExternalID(string)

	// required fields for SMPP
	GatewayID() string // External ID from mGage for tracking MsgStatus
	SetGatewayID(string)
	CarrierID() string // External ID from service behind mGage for tracking MsgStatus
	SetCarrierID(string)

	Status() MsgStatusValue
	SetStatus(MsgStatusValue)

	Logs() []*ChannelLog
	AddLog(log *ChannelLog)
}
