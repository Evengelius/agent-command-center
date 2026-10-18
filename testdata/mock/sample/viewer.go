// Package sample is the file the file viewer's golden frames show (plan 17):
// a short Go program with comments, strings, numbers and an edit to mark.
package sample

import (
	"fmt"
	"strings"
)

// Colour is a named colour of the palette.
type Colour struct {
	Name string
	RGB  uint32
}

// palette holds the colours the dashboard draws with.
var palette = []Colour{
	{"title", 0x7aa2f7},
	{"text", 0xc0caf5},
	{"grey", 0x565f89},
	{"attention", 0xe0af68},
}

// Find returns the colour called name, and whether there is one.
func Find(name string) (Colour, bool) {
	for _, c := range palette {
		if strings.EqualFold(c.Name, name) {
			return c, true
		}
	}
	return Colour{}, false
}

// Hex is the colour as "#rrggbb".
func (c Colour) Hex() string {
	return fmt.Sprintf("#%06x", c.RGB)
}

// Names lists the palette's colour names, in order.
func Names() []string {
	out := make([]string, 0, len(palette))
	for _, c := range palette {
		out = append(out, c.Name)
	}
	return out
}

// Brighter is the colour with each channel raised by step, at most 255.
func (c Colour) Brighter(step uint32) Colour {
	r, g, b := c.RGB>>16&0xff, c.RGB>>8&0xff, c.RGB&0xff
	up := func(v uint32) uint32 { return min(v+step, 255) }
	return Colour{c.Name, up(r)<<16 | up(g)<<8 | up(b)}
}

// Describe is one line about the colour, for the details view.
func Describe(c Colour) string {
	return fmt.Sprintf("%-10s %s", c.Name, c.Hex())
}
