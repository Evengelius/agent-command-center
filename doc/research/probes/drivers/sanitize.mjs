// Removes personal data from probe recordings before they enter the repository.
// Usage: PROBE_USER=<windows user> PROBE_HOST=<host name> PROBE_ORG=<organisation prefix> \
//          node sanitize.mjs <in-file> <out-file>
import { readFileSync, writeFileSync } from "node:fs";

const [inFile, outFile] = process.argv.slice(2);
const { PROBE_USER: U, PROBE_HOST: H, PROBE_ORG: G } = process.env;
if (!U || !H || !G) throw new Error("set PROBE_USER, PROBE_HOST and PROBE_ORG");
const esc = (x) => x.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
const SEP = "(?:\\\\+|/)+";
const REDACT_KEYS = new Set(["email", "organization", "installationId", "serverName"]);

const rules = [
  [new RegExp(`C:${SEP}Users${SEP}${esc(U)}${SEP}AppData${SEP}Local${SEP}Temp${SEP}claude${SEP}c--javadev-projets-agent-command-center${SEP}[0-9a-f-]{36}${SEP}scratchpad`, "gi"), "<probe-root>"],
  [new RegExp(`C--Users-${esc(U)}-AppData-Local-Temp-claude-c--javadev-projets-agent-command-center-[0-9a-f-]{36}-scratchpad`, "g"), "<probe-root-slug>"],
  [new RegExp(`(?:/c|C:)${SEP}Users${SEP}${esc(U)}`, "gi"), "<HOME>"],
  [new RegExp(esc(U), "gi"), "<user>"],
  [new RegExp(esc(H), "g"), "<host>"],
  [/[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[a-z]{2,}/g, "<email>"],
  [new RegExp(`${esc(G)}[^"\\\\]*`, "g"), "<organization>"],
  [/RateLimitResetCredit_[0-9a-f]{8,}/g, "RateLimitResetCredit_<redacted>"],
];
const cleanString = (s) => rules.reduce((acc, [re, to]) => acc.replace(re, to), s);

const walk = (v, key) => {
  if (typeof v === "string") {
    if (REDACT_KEYS.has(key)) return "<redacted>";
    if (key === "signature") return "<signature>";
    return cleanString(v);
  }
  if (Array.isArray(v)) return v.map((x) => walk(x, key));
  if (v && typeof v === "object")
    return Object.fromEntries(Object.entries(v).map(([k, x]) => [k, walk(x, k)]));
  return v;
};

// Streamed deltas can split a name across chunks ("...\Users\amol" + "imom..."),
// which no rule sees in one string. Each stream's chunks are joined, cleaned,
// and the result written back into the same chunks: a match goes into the chunk
// where it starts, and the chunks it covered keep only what follows it.
const deltaRef = (rec) => {
  const m = rec && rec.msg;
  if (!m) return null;
  const d = m.event && m.event.type === "content_block_delta" && m.event.delta;
  if (d) {
    const field = ["text", "thinking", "partial_json"].find((f) => typeof d[f] === "string");
    if (field) return { key: `cc|${m.parent_tool_use_id || ""}|${m.event.index}|${field}`, obj: d, field };
  }
  if (m.params && typeof m.params.delta === "string" && /\/delta$/.test(m.method || ""))
    return { key: `cx|${m.params.threadId || ""}|${m.params.itemId || ""}`, obj: m.params, field: "delta" };
  return null;
};
const cleanChunks = (refs) => {
  for (const [re, to] of rules) {
    const joined = refs.map((r) => r.obj[r.field]).join("");
    const matches = [...joined.matchAll(re)];
    for (const match of matches.reverse()) {
      const ms = match.index, me = ms + match[0].length;
      let at = 0;
      for (const r of refs) {
        const s = r.obj[r.field], cs = at, ce = at + s.length;
        at = ce;
        if (ce <= ms || cs >= me) continue;
        const head = ms >= cs ? s.slice(0, ms - cs) + to : "";
        const tail = me < ce ? s.slice(me - cs) : "";
        r.obj[r.field] = head + tail;
      }
    }
  }
};
const cleanStreams = (recs) => {
  const streams = new Map();
  for (const rec of recs) {
    const m = rec && rec.msg;
    if (m && m.event && m.event.type === "message_start") {
      const p = `cc|${m.parent_tool_use_id || ""}|`;
      for (const k of [...streams.keys()]) if (k.startsWith(p)) { cleanChunks(streams.get(k)); streams.delete(k); }
    }
    const ref = deltaRef(rec);
    if (!ref) continue;
    if (!streams.has(ref.key)) streams.set(ref.key, []);
    streams.get(ref.key).push(ref);
  }
  for (const refs of streams.values()) cleanChunks(refs);
  return recs;
};

const text = readFileSync(inFile, "utf8").replace(/\r\n/g, "\n");
let out;
if (/\.jsonl$/.test(inFile)) {
  const lines = text.split("\n").filter(Boolean).map((l) => {
    try { return walk(JSON.parse(l), ""); } catch { return cleanString(l); }
  });
  out = cleanStreams(lines).map((v) => (typeof v === "string" ? v : JSON.stringify(v))).join("\n") + "\n";
} else if (/\.json$/.test(inFile)) {
  out = JSON.stringify(walk(JSON.parse(text), ""), null, 1) + "\n";
} else {
  out = cleanString(text);
}
writeFileSync(outFile, out);
