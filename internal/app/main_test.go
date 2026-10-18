package app

import (
	"os"
	"sync"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/render"
)

// The harness colours outside its synctest bubble. Chroma gives each match
// of regexp2, its regular expressions, a 250 ms timeout, which regexp2
// checks against a package-wide clock goroutine started on first use and
// stopped about a second and a half after the last match; started inside a
// bubble, that goroutine outlives the test and the bubble panics. Here every
// colouring runs on one goroutine started before the tests, outside every
// bubble, through channels made there too (plan 17, Highlighting).
type hlReq struct {
	name     string
	lines    []string
	from, to int
}

var (
	hlMu      sync.Mutex
	hlReqs    = make(chan hlReq)
	hlReplies = make(chan render.Highlighted)
)

func TestMain(m *testing.M) {
	go func() {
		for r := range hlReqs {
			hlReplies <- render.Highlight(r.name, r.lines, r.from, r.to)
		}
	}()
	os.Exit(m.Run())
}

// outsideHighlight is render.Highlight run on the goroutine above.
func outsideHighlight(name string, lines []string, from, to int) render.Highlighted {
	hlMu.Lock()
	defer hlMu.Unlock()
	hlReqs <- hlReq{name, lines, from, to}
	return <-hlReplies
}
