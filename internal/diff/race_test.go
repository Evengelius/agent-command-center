//go:build race

package diff

// raceEnabled lengthens the time limits: the race detector slows the search.
const raceEnabled = true
