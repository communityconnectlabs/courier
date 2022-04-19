package utils_test

import (
	"github.com/nyaruka/courier/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNullMap_Scan(t *testing.T) {
	var validMap map[string]interface{}
	nullMap := utils.NewNullMap(validMap)

	assert.Equal(t, true, nullMap.Valid)

	dataMap := map[string]interface{}{"age": 30, "name": "John"}
	err := nullMap.Scan(dataMap)

	assert.Errorf(t, err, "Incompatible type for NullDict")

	jsonData := []byte(`{"age": 20, "name": "john"}`)
	err = nullMap.Scan(jsonData)
	assert.NoError(t, err)
}

func TestNullMap_Value(t *testing.T) {
	nullMap := utils.NullMap{}
	jsonData := []byte(`{"age": 20, "name": "john"}`)
	err := nullMap.Scan(jsonData)
	assert.NoError(t, err)
	assert.Equal(t, true, nullMap.Valid)
	assert.Equal(t, 2, len(nullMap.Map))

	value, err := nullMap.Value()
	assert.NoError(t, err)
	value2, err := nullMap.MarshalJSON()

	assert.NoError(t, err)
	assert.Equal(t, value2, value)

	nullMap.UnmarshalJSON([]byte(`{"age": 31}`))

	assert.Equal(t, 1, len(nullMap.Map))
}
