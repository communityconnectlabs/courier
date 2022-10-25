package handlers_test

import (
	"github.com/nyaruka/courier/handlers"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDecodeAndValidateForm(t *testing.T) {
	type formStruct struct {
		FirstName []string
		LastName  []string
		URL       []string
	}

	formData := url.Values{
		"FirstName": []string{"John"},
		"LastName":  []string{"Doe"},
		"URL":       []string{"example.com/john-doe"},
	}
	r := httptest.NewRequest("GET", "https://example.com", nil)

	err := handlers.DecodeAndValidateForm("", r)
	assert.Errorf(t, err, "schema: interface must be a pointer to struct")

	r.Form = formData
	form := &formStruct{}
	err = handlers.DecodeAndValidateForm(form, r)
	assert.NoError(t, err)

	assert.Equal(t, "John", form.FirstName[0])
}

func TestDecodeAndValidateJSON(t *testing.T) {
	type jsonStruct struct {
		FirstName string `json:"first_name"`
	}
	jsonData := &jsonStruct{}

	r := httptest.NewRequest("GET", "https://example.com", nil)
	err := handlers.DecodeAndValidateJSON(jsonData, r)

	assert.Errorf(t, err, "unable to parse request JSON: unexpected end of JSON input")

	jsonBody := `{"first_name": "John"}`
	b := strings.NewReader(jsonBody)
	r = httptest.NewRequest("GET", "https://example.com", b)

	err = handlers.DecodeAndValidateJSON(jsonData, r)
	assert.NoError(t, err)

	assert.Equal(t, "John", jsonData.FirstName)
}

func TestDecodeAndValidateXML(t *testing.T) {
	type xmlStruct struct {
		FirstName string `xml:"FirstName"`
	}
	xmlData := &xmlStruct{}

	r := httptest.NewRequest("GET", "https://example.com", nil)
	err := handlers.DecodeAndValidateXML(xmlData, r)

	assert.Errorf(t, err, "unable to parse request XML: EOF")

	xmlBody := `<?xml version="1.0" encoding="UTF-8"?>
	<Body>
		<FirstName>John</FirstName>
	</Body>`
	b := strings.NewReader(xmlBody)
	r = httptest.NewRequest("GET", "https://example.com", b)

	err = handlers.DecodeAndValidateXML(xmlData, r)
	assert.NoError(t, err)

	assert.Equal(t, "John", xmlData.FirstName)
}

func TestReadBody(t *testing.T) {
	text := "this is a short text"
	b := strings.NewReader(text)
	r := httptest.NewRequest("GET", "https://example.com", b)
	body, err := handlers.ReadBody(r, 5)
	assert.NoError(t, err)
	assert.Equal(t, int64(20), r.ContentLength)
	assert.Equal(t, 5, len(body))
}
