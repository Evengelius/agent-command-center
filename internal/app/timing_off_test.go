//go:build !timing

package app

// timingBuild is false outside the timing step: perf_test.go's tests still
// run for their other checks, and their times are logged, not judged (plan
// 21, item 2; doc/evaluation.md, Keystroke timing).
const timingBuild = false
