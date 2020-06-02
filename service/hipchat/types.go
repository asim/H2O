package hipchat

import (
	"fmt"

	"github.com/HailoOSS/go-hailo-lib/multierror"
)

// Notifier represents something that can send notifications
type Notifier interface {
	Notify(msg *Message) error
}

// Message represents some message we send
type Message struct {
	Room   string // Room name or ID
	From   string // From is who we are sending the message from
	Body   string // Body is the message content
	Format Format // Format that the body is supplied in; text or html
	Notify bool   // Notify indicates we want to make the Hipchat icon bounce around like mad and annoy people
	Colour Colour // Colour is the background colour of the announcement
}

// Valid tests if the message is valid for sending
func (msg *Message) Valid() error {
	errs := multierror.New()
	if msg.Room == "" {
		errs.Add(fmt.Errorf("'Room' cannot be empty"))
	}
	if msg.From == "" {
		errs.Add(fmt.Errorf("'From' cannot be empty"))
	}
	if !msg.Format.Valid() {
		errs.Add(fmt.Errorf("'Format' is invalid"))
	}
	if !msg.Colour.Valid() {
		errs.Add(fmt.Errorf("'Colour' is invalid"))
	}
	if len(msg.From) > 15 {
		errs.Add(fmt.Errorf("From cannot be more than 15 characters"))
	}
	if errs.AnyErrors() {
		return errs
	}
	return nil
}

type Format string

const (
	FormatHtml Format = "html"
	FormatText        = "text"
)

var ValidFormats = map[Format]bool{
	FormatHtml: true,
	FormatText: true,
}

// Valid tests if this format is valid
func (f Format) Valid() bool {
	return ValidFormats[f]
}

type Colour string

const (
	ColourYellow Colour = "yellow"
	ColourRed           = "red"
	ColourGreen         = "green"
	ColourPurple        = "purple"
	ColourGrey          = "gray"
	ColourRandom        = "random"
)

var ValidColours = map[Colour]bool{
	ColourYellow: true,
	ColourRed:    true,
	ColourGreen:  true,
	ColourPurple: true,
	ColourGrey:   true,
	ColourRandom: true,
}

// Valid tests if this colour is valid
func (c Colour) Valid() bool {
	return ValidColours[c]
}
