package attributes

import (
	"net/http"
	"testing"

	"github.com/roadrunner-server/http/v4/attributes"
	"github.com/stretchr/testify/assert"
)

func TestAllAttributes(t *testing.T) {
	r := &http.Request{}
	r = attributes.Init(r)

	err := attributes.Set(r, "key", "value")
	if err != nil {
		t.Errorf("error during the Set: error %v", err)
	}

	assert.Equal(t, attributes.All(r), map[string]any{
		"key": "value",
	})
}

func TestAllAttributesNone(t *testing.T) {
	r := &http.Request{}
	r = attributes.Init(r)

	assert.Equal(t, attributes.All(r), map[string]any{})
}

func TestAllAttributesNone2(t *testing.T) {
	r := &http.Request{}

	assert.Equal(t, attributes.All(r), map[string]any{})
}

func TestGetAttribute(t *testing.T) {
	r := &http.Request{}
	r = attributes.Init(r)

	err := attributes.Set(r, "key", "value")
	if err != nil {
		t.Errorf("error during the Set: error %v", err)
	}
	assert.Equal(t, attributes.Get(r, "key"), "value")
}

func TestGetAttributeNone(t *testing.T) {
	r := &http.Request{}
	r = attributes.Init(r)

	assert.Equal(t, attributes.Get(r, "key"), nil)
}

func TestGetAttributeNone2(t *testing.T) {
	r := &http.Request{}

	assert.Equal(t, attributes.Get(r, "key"), nil)
}

func TestSetAttribute(t *testing.T) {
	r := &http.Request{}
	r = attributes.Init(r)

	err := attributes.Set(r, "key", "value")
	if err != nil {
		t.Errorf("error during the Set: error %v", err)
	}
	assert.Equal(t, attributes.Get(r, "key"), "value")
}

func TestSetAttributeNone(t *testing.T) {
	r := &http.Request{}
	err := attributes.Set(r, "key", "value")
	assert.Error(t, err)
	assert.Equal(t, attributes.Get(r, "key"), nil)
}
