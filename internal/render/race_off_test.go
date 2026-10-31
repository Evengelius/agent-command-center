//go:build !race

package render

// raceBuild is false: TestMarkdownCacheBytes judges a frame's bytes.
const raceBuild = false
