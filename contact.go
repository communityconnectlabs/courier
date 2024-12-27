package courier

import (
	"strings"

	"github.com/gofrs/uuid"
	"github.com/nyaruka/null"
)

// ContactUUID is our typing of a contact's UUID
type ContactUUID struct {
	uuid.UUID
}

// NilContactUUID is our nil value for contact UUIDs
var NilContactUUID = ContactUUID{uuid.Nil}

// NewContactUUID creates a new ContactUUID for the passed in string
func NewContactUUID(u string) (ContactUUID, error) {
	contactUUID, err := uuid.FromString(strings.ToLower(u))
	if err != nil {
		return NilContactUUID, err
	}
	return ContactUUID{contactUUID}, nil
}

//-----------------------------------------------------------------------------
// Contact Interface
//-----------------------------------------------------------------------------

// Contact defines the attributes on a contact, for our purposes that is just a contact UUID
type Contact interface {
	UUID() ContactUUID
	Language() null.String
}

// ContactFieldUUID is our typing of a contact field's UUID
type ContactFieldUUID struct {
	uuid.UUID
}

// ContactField defines the attributes on a contact field, for our purposes that is just a contact field UUID
type ContactField interface {
	UUID() ContactFieldUUID
}
