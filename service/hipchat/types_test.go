package hipchat

import (
	"testing"
)

func TestFormatValid(t *testing.T) {
	testCases := []struct {
		f     Format
		valid bool
	}{
		{FormatHtml, true},
		{FormatText, true},
		{Format("FOO"), false},
		{Format(""), false},
		{Format("Text"), false},
	}

	for _, tc := range testCases {
		if tc.valid != tc.f.Valid() {
			t.Errorf("Expected %v to return %v", tc.f, tc.valid)
		}
	}
}

func TestColourValid(t *testing.T) {
	testCases := []struct {
		c     Colour
		valid bool
	}{
		{ColourGreen, true},
		{ColourGrey, true},
		{ColourPurple, true},
		{ColourRandom, true},
		{ColourRed, true},
		{ColourYellow, true},
		{Colour("FOO"), false},
		{Colour(""), false},
		{Colour("Text"), false},
	}

	for _, tc := range testCases {
		if tc.valid != tc.c.Valid() {
			t.Errorf("Expected %v to return %v", tc.c, tc.valid)
		}
	}
}

func TestValid(t *testing.T) {
	testCases := []struct {
		msg   *Message
		valid bool
	}{
		{
			msg: &Message{
				Room:   "Cruft",
				From:   "Hipster",
				Body:   "Go service layer Hipchat client unit testing (likeaboss)",
				Format: FormatText,
				Notify: false,
				Colour: ColourGreen,
			},
			valid: true,
		},
		{
			msg: &Message{
				From:   "Hipster",
				Body:   "Go service layer Hipchat client unit testing (likeaboss)",
				Format: FormatText,
				Notify: false,
				Colour: ColourGreen,
			},
			valid: false,
		},
		{
			msg: &Message{
				Room:   "Cruft",
				Body:   "Go service layer Hipchat client unit testing (likeaboss)",
				Format: FormatText,
				Notify: false,
				Colour: ColourGreen,
			},
			valid: false,
		},
		{
			msg: &Message{
				Room:   "Cruft",
				From:   "Hipster",
				Body:   "Go service layer Hipchat client unit testing (likeaboss)",
				Format: Format("Foo"),
				Notify: false,
				Colour: ColourGreen,
			},
			valid: false,
		},
		{
			msg: &Message{
				Room:   "Cruft",
				From:   "Hipster",
				Body:   "Go service layer Hipchat client unit testing (likeaboss)",
				Format: FormatText,
				Notify: false,
				Colour: Colour("Magento"),
			},
			valid: false,
		},
	}

	for _, tc := range testCases {
		errs := tc.msg.Valid()
		if errs != nil && tc.valid {
			t.Errorf("Expecting %v to be valid, got err %v", tc.msg, errs.Error())
		} else if errs == nil && !tc.valid {
			t.Errorf("Not expecting %v to be valid", tc.msg)
		}
	}
}
