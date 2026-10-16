//go:build race

package app

// raceBuild is true under -race, whose instrumentation slows the code
// several times over: the timing targets (perf_test.go) are not judged then.
const raceBuild = true
