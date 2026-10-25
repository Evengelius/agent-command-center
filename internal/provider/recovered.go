package provider

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// Recovered is a panic recovered in a goroutine of the dashboard's own, as
// text: its value, and the function, file and line it came from. The
// goroutine shows it as an error and ends its session in order, rather than
// ending the program with the terminal in its modes (plan 48, design D5;
// audit finding 6). It is called by the deferred function that recovered v.
func Recovered(v any) string {
	pcs := make([]uintptr, 64)
	frames := runtime.CallersFrames(pcs[:runtime.Callers(1, pcs)])
	panicking := false
	for {
		f, more := frames.Next()
		if panicking && !strings.HasPrefix(f.Function, "runtime.") {
			return fmt.Sprintf("panic: %v (at %s, %s:%d)", v, f.Function[strings.LastIndex(f.Function, "/")+1:], filepath.Base(f.File), f.Line)
		}
		panicking = panicking || f.Function == "runtime.gopanic"
		if !more {
			return fmt.Sprintf("panic: %v", v)
		}
	}
}
