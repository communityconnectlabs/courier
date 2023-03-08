package courier

import (
	"database/sql/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestChannelID(t *testing.T) {
	channelID := NewChannelID(101)

	value, err := channelID.Value()
	require.NoError(t, err)
	assert.Equal(t, int64(101), value.(driver.Value))
	val, _ := channelID.MarshalJSON()
	assert.Equal(t, []byte(`101`), val)

	channelID2 := NewChannelID(0)
	val, _ = channelID2.MarshalJSON()
	assert.Equal(t, []byte(`null`), val)

	err = channelID2.UnmarshalJSON([]byte(`10`))
	require.NoError(t, err)

	assert.Equal(t, ChannelID(10), channelID2)
}
