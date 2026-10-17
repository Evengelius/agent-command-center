package confcheck

// Key-level diff of ~/.claude.json, the owner's approval, 2026-10-17
// (doc/reliability.md, Configuration untouched). For this test tool only:
// key names and digests of compacted values are kept, never a value; keys
// with a sensitive name are not even hashed.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var sensitiveKey = regexp.MustCompile(`(?i)oauth|token|secret|password|credential|api_?key|auth|account|email|session`)

var stateNames = map[string]bool{
	"numStartups": true, "firstStartTime": true, "userID": true, "machineID": true,
	"migrationVersion": true, "feedbackSurveyState": true, "pluginUsage": true,
	"skillUsage": true, "seenNotifications": true, "announcementImpressions": true,
	"clientDataCacheSlots": true, "metricsStatusCacheByPrincipal": true,
	"modelAccessCache": true, "remoteControlUpsellSeenCount": true,
	"installMethod": true, "hasCompletedOnboarding": true,
}

var statePrefixes = []string{"cached", "tips", "tip", "last", "has", "unpin", "official",
	"additionalModel", "autoCompact", "orgModel", "changelog", "closedIssues"}

var stateSuffixes = []string{"MigrationComplete", "Cache", "At"}

// Sub-keys of a project entry that are configuration; the others are state.
var projectConfigKeys = map[string]bool{
	"allowedTools": true, "mcpServers": true, "enabledMcpjsonServers": true,
	"disabledMcpjsonServers": true, "mcpContextUris": true,
	"hasTrustDialogAccepted": true, "ignorePatterns": true,
}

func isStateKey(k string) bool {
	if stateNames[k] {
		return true
	}
	for _, p := range statePrefixes {
		if strings.HasPrefix(k, p) {
			return true
		}
	}
	for _, s := range stateSuffixes {
		if strings.HasSuffix(k, s) {
			return true
		}
	}
	return false
}

func digestRaw(raw json.RawMessage) string {
	var b bytes.Buffer
	if err := json.Compact(&b, raw); err != nil {
		b.Reset()
		b.Write(raw)
	}
	sum := sha256.Sum256(b.Bytes())
	return hex.EncodeToString(sum[:])
}

// keyDigests decodes data; ok is false when it is not a JSON object.
func keyDigests(data []byte) (keys map[string]string, projects map[string]map[string]string, skipped []string, ok bool) {
	var top map[string]json.RawMessage
	if json.Unmarshal(data, &top) != nil || top == nil {
		return nil, nil, nil, false
	}
	keys = map[string]string{}
	for k, v := range top {
		if sensitiveKey.MatchString(k) {
			skipped = append(skipped, k)
			continue
		}
		if k == "projects" {
			var ps map[string]map[string]json.RawMessage
			if json.Unmarshal(v, &ps) == nil && ps != nil {
				projects = map[string]map[string]string{}
				for path, sub := range ps {
					m := map[string]string{}
					for sk, sv := range sub {
						if !sensitiveKey.MatchString(sk) {
							m[sk] = digestRaw(sv)
						}
					}
					projects[path] = m
				}
				continue
			}
		}
		keys[k] = digestRaw(v)
	}
	sort.Strings(skipped)
	return keys, projects, skipped, true
}

func normPath(p string) string {
	p = strings.ToLower(strings.ReplaceAll(p, `\`, "/"))
	for len(p) > 1 && strings.HasSuffix(p, "/") {
		p = strings.TrimSuffix(p, "/")
	}
	return p
}

func changedKeys(b, a map[string]string) []string {
	set := map[string]bool{}
	for k, v := range b {
		if av, ok := a[k]; !ok || av != v {
			set[k] = true
		}
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			set[k] = true
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// diffClaudeJSON returns report lines and the verdict inputs.
func diffClaudeJSON(name string, be, ae Entry, project string, cliRan bool) (lines []string, fail, mixed bool) {
	line := func(f string, args ...any) {
		lines = append(lines, fmt.Sprintf("%s  %s  ", ClassMixed, name)+fmt.Sprintf(f, args...))
	}
	changed := 0
	for _, k := range changedKeys(be.Keys, ae.Keys) {
		changed++
		if isStateKey(k) {
			line("key %s  changed (state)", k)
			if !cliRan {
				fail = true
			}
		} else {
			line("key %s  changed (config)", k)
			fail = true
		}
	}
	sens := map[string]bool{}
	for _, k := range be.Skipped {
		sens[k] = true
	}
	for _, k := range ae.Skipped {
		sens[k] = true
	}
	var sk []string
	for k := range sens {
		sk = append(sk, k)
	}
	sort.Strings(sk)
	for _, k := range sk {
		line("key %s  not compared (sensitive name)", k)
	}

	paths := map[string]bool{}
	for p := range be.Projects {
		paths[p] = true
	}
	for p := range ae.Projects {
		paths[p] = true
	}
	var ps []string
	for p := range paths {
		ps = append(ps, p)
	}
	sort.Strings(ps)
	for _, p := range ps {
		ch := changedKeys(be.Projects[p], ae.Projects[p])
		if len(ch) == 0 {
			continue
		}
		changed++
		if project == "" || normPath(p) != normPath(project) {
			line("projects[%s]  changed (another project)", p)
			mixed = true
			continue
		}
		for _, k := range ch {
			if projectConfigKeys[k] {
				line("projects[%s].%s  changed (config)", p, k)
				fail = true
			} else {
				line("projects[%s].%s  changed (state)", p, k)
				if !cliRan {
					fail = true
				}
			}
		}
	}
	// The digest changed but no compared key did: the change is in a key not
	// compared or in formatting, so it is not verified, never a silent pass.
	if changed == 0 {
		line("no compared key changed (a key not compared, or formatting)")
		mixed = true
	}
	return lines, fail, mixed
}
