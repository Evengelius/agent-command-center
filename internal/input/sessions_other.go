//go:build !windows

package input

// held: outside Windows an open file cannot be told from a closed one
// without advisory locks, so the sessions column is N/A (plan 16).
func held(string) (bool, error) { return false, ErrSessionsUnsupported }
