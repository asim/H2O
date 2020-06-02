package weberrors

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test400(t *testing.T) {
	e := New()
	if e.HasErrors() {
		t.Fail()
	}
	e.SetC("name", "notUnique")
	if !e.HasErrors() {
		t.Fail()
	}
	if len(e.Context()) != 2 {
		t.Fail()
	}
	if len(e.Description()) == 0 {
		t.Fail()
	}
	if e.HttpCode() != 400 {
		t.Fail()
	}
	if e.Type() != BadRequest {
		t.Fail()
	}
	e.UnsetC("name")
	if e.HasErrors() {
		t.Fail()
	}
}

func Test500(t *testing.T) {
	e := New()
	desc := "Failed 'cause of whatever."
	e.SetD(desc)
	if !e.HasErrors() {
		t.Fail()
	}
	if e.HttpCode() != 500 {
		t.Fail()
	}
	if e.Type() != InternalServerError {
		t.Fail()
	}
}

func TestAddContext(t *testing.T) {
	e1 := New()
	e1.AddContext("key1", "value1", "key2", "value2")
	assert.Equal(t, map[string]string{
		"key1": "value1",
		"key2": "value2",
	}, e1.c)

	e2 := New()
	e2.AddContext("key1", "value1")
	assert.Equal(t, map[string]string{
		"key1": "value1",
	}, e2.c)

	e3 := New()
	e3.AddContext()
	assert.Equal(t, map[string]string{}, e3.c)

	e4 := New()
	e4.AddContext("key1")
	assert.Equal(t, map[string]string{}, e4.c)

	e5 := New()
	e5.AddContext("key1", "value1", "key2")
	assert.Equal(t, map[string]string{
		"key1": "value1",
	}, e5.c)
}
