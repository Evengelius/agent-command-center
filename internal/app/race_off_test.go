//go:build !race

package app

// raceBuild is false: the timing targets (perf_test.go) are judged.
const raceBuild = false
