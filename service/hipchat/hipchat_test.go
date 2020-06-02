// +build integration

package hipchat

import (
	"testing"
)

func TestViaHttp(t *testing.T) {
	msg := &Message{
		Room:   "Cruft",
		From:   "Hipster",
		Body:   "Go service layer Hipchat client unit testing (likeaboss)",
		Format: FormatText,
		Notify: false,
		Colour: ColourGreen,
	}
	err := Notify(msg)
	if err != nil {
		t.Fatalf("Error notifying: %v", err)
	}
}
