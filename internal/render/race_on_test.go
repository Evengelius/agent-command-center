//go:build race

package render

// raceBuild is true under -race, whose instrumentation changes what the code
// allocates: TestMarkdownCacheBytes does not judge a frame's bytes then.
const raceBuild = true
