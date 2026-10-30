// Plan 69's probe P69a (plan.md, Plan 69, criterion 69.1; run within the
// proposed quota as the owner's design review of 2026-10-29 asked): Claude
// Code resumed in a new process. The dashboard's arguments
// (internal/provider/claude/session.go, args) and handshake (initialize,
// then get_settings and the user message "/usage", which Claude Code answers
// locally, with no model call: P24u), with CLAUDE_CODE_EMIT_SESSION_STATE_EVENTS
// as the dashboard sets it; every CLAUDE* and ANTHROPIC_* variable of this
// shell dropped, as P39c's, and Haiku as the per-process model
// (ANTHROPIC_MODEL=haiku, the standing decision). Three processes in turn:
//   new      a new session; after the handshake one message with a code word
//            (one model call);
//   resume   --resume <the new session's id>; nothing but initialize for
//            4 s, to see what a resumed process sends unasked, then the
//            handshake and one message asking for the code word (one model
//            call);
//   unknown  --resume <a random UUID no session has>; the handshake only
//            (no model call).
// Every tool request is denied, any other request answered as not handled.
// A process ends with stdin closed 1.5 s after its last result, or killed at
// 120 s. Records every line with the process's name; the initialize and
// get_settings responses and system/init as filtered views (no account,
// commands, agents, skills, plugins or MCP names); home, user, host and work
// paths replaced.
// Usage: node p69-claude-resume.mjs <workdir> <outfile> <codeword>
import { spawn, execFileSync } from "node:child_process";
import { appendFileSync, writeFileSync } from "node:fs";
import { randomUUID } from "node:crypto";

const [workdir, outfile, word] = process.argv.slice(2);
if (!workdir || !outfile || !word) throw new Error("usage: node p69-claude-resume.mjs <workdir> <outfile> <codeword>");
const t0 = Date.now();
writeFileSync(outfile, "");

const esc = (x) => x.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
const home = process.env.USERPROFILE ?? "";
const hides = [
  [JSON.stringify(workdir).slice(1, -1), "<probe-work>"],
  [workdir, "<probe-work>"],
  [workdir.replaceAll("\\", "/"), "<probe-work>"],
  [JSON.stringify(home).slice(1, -1), "<HOME>"],
  [home, "<HOME>"],
  [home.replaceAll("\\", "/"), "<HOME>"],
  [process.env.USERNAME ?? "", "<user>"],
  [process.env.COMPUTERNAME ?? "", "<host>"],
].filter(([s]) => s.length >= 3);
const clean = (s) => hides.reduce((a, [from, to]) => a.replace(new RegExp(esc(from), "gi"), to), s)
  .replace(/[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[a-z]{2,}/g, "<email>");
const rec = (proc, dir, obj) =>
  appendFileSync(outfile, clean(JSON.stringify({ t_ms: Date.now() - t0, proc, dir, msg: obj })) + "\n");

const base = ["-p", "--input-format", "stream-json", "--output-format", "stream-json",
  "--verbose", "--include-partial-messages", "--forward-subagent-text",
  "--permission-prompt-tool", "stdio", "--permission-mode", "default"];
const env = Object.fromEntries(Object.entries(process.env).filter(([k]) => !/^(CLAUDE|ANTHROPIC_)/i.test(k)));
env.CLAUDE_CODE_EMIT_SESSION_STATE_EVENTS = "1";
env.ANTHROPIC_MODEL = "haiku";
rec("driver", "meta", { cwd: "<probe-work>", word,
  dropped: Object.keys(process.env).filter((k) => !(k in env)).sort(),
  set: { CLAUDE_CODE_EMIT_SESSION_STATE_EVENTS: "1", ANTHROPIC_MODEL: "haiku" } });

const view = (subtype, r) => {
  if (!r || typeof r !== "object") return r;
  if (subtype === "initialize") {
    return { claude_code_version: r.claude_code_version, current_permission_mode: r.current_permission_mode,
      account: { subscriptionType: r.account?.subscriptionType, apiProvider: r.account?.apiProvider },
      session: Object.fromEntries(Object.entries(r).filter(([k]) => /session/i.test(k))), keys: Object.keys(r) };
  }
  if (subtype === "get_settings") return { applied: r.applied, keys: Object.keys(r) };
  return r;
};
const initView = (m) => ({ type: m.type, subtype: m.subtype, session_id: m.session_id, model: m.model, cwd: m.cwd,
  permissionMode: m.permissionMode, claude_code_version: m.claude_code_version, apiKeySource: m.apiKeySource,
  tools: Array.isArray(m.tools) ? m.tools.length : m.tools, keys: Object.keys(m) });

// run starts one process and drives it: afterInit is called with the
// process's helpers once initialize is answered; onResult with the count of
// results so far. Resolves with what the process reported, once it exits.
const run = (proc, extra, afterInit, onResult) => new Promise((resolve) => {
  const argv = [...base, ...extra];
  rec(proc, "meta", { argv: ["claude", ...argv] });
  const child = spawn("claude", argv, { cwd: workdir, env, stdio: ["pipe", "pipe", "pipe"] });
  const seen = { sessionIds: [], results: 0, texts: [], beforeFirstUser: [], userSent: false };
  let n = 0;
  const sent = new Map();
  const send = (obj) => { rec(proc, "in", obj); child.stdin.write(JSON.stringify(obj) + "\n"); };
  const control = (request) => { const id = `req_${++n}`; sent.set(id, request.subtype); send({ type: "control_request", request_id: id, request }); };
  const user = (text) => { seen.userSent = true; send({ type: "user", message: { role: "user", content: text }, parent_tool_use_id: null, session_id: "default" }); };
  let ending = false;
  const end = () => { if (!ending) { ending = true; setTimeout(() => child.stdin.end(), 1500); } };
  const handshake = () => { control({ subtype: "get_settings" }); user("/usage"); };
  const h = { control, user, end, handshake };
  let buf = "";
  child.stdout.on("data", (d) => {
    buf += d.toString("utf8");
    let i;
    while ((i = buf.indexOf("\n")) >= 0) {
      const line = buf.slice(0, i); buf = buf.slice(i + 1);
      if (!line.trim()) continue;
      let m; try { m = JSON.parse(line); } catch { rec(proc, "out-raw", line.slice(0, 2000)); continue; }
      if (m.session_id && m.session_id !== "default" && !seen.sessionIds.includes(m.session_id)) seen.sessionIds.push(m.session_id);
      if (!seen.userSent) seen.beforeFirstUser.push([m.type, m.subtype].filter(Boolean).join("/"));
      if (m.type === "control_response") {
        const id = m.response?.request_id, subtype = sent.get(id);
        sent.delete(id);
        rec(proc, "out", { type: m.type, response: { subtype: m.response?.subtype, request_id: id, error: m.response?.error,
          response: view(subtype, m.response?.response) } });
        if (subtype === "initialize") afterInit(h);
        continue;
      }
      if (m.type === "system" && m.subtype === "init") { rec(proc, "out", initView(m)); continue; }
      rec(proc, "out", m);
      if (m.type === "control_request") {
        if (m.request?.subtype === "can_use_tool") send({ type: "control_response", response: { subtype: "success", request_id: m.request_id, response: { behavior: "deny", message: "probe: no tool in this check" } } });
        else send({ type: "control_response", response: { subtype: "error", request_id: m.request_id, error: "probe: not handled" } });
      }
      if (m.type === "assistant" && !m.parent_tool_use_id) {
        for (const c of m.message?.content ?? []) if (c.type === "text") seen.texts.push(c.text);
      }
      if (m.type === "result") { seen.results++; onResult(h, seen.results, m); }
    }
  });
  child.stderr.on("data", (d) => rec(proc, "stderr", d.toString("utf8").slice(0, 2000)));
  const killer = setTimeout(() => {
    rec(proc, "timeout", {});
    try { execFileSync("taskkill", ["/T", "/F", "/PID", String(child.pid)]); } catch { child.kill(); }
  }, 120000);
  child.on("exit", (code, signal) => {
    clearTimeout(killer);
    rec(proc, "exit", { code, signal, ...seen });
    resolve({ code, ...seen });
  });
  control({ subtype: "initialize", hooks: null });
});

const ask = `Remember this code word for later in this conversation: ${word}. Reply with the single word OK, and nothing else.`;
const recall = "What is the code word I asked you to remember earlier in this conversation? Reply with the code word only.";

// new: the handshake, then the code word once "/usage" is answered.
const a = await run("new", [], (h) => h.handshake(), (h, n) => { if (n === 1) h.user(ask); else h.end(); });
const id = a.sessionIds[0];
if (!id) { rec("driver", "stop", { reason: "the new session reported no session id" }); process.exit(0); }

// resume: 4 s of initialize alone, then the handshake and the question.
const b = await run("resume", ["--resume", id], (h) => setTimeout(() => h.handshake(), 4000),
  (h, n) => { if (n === 1) h.user(recall); else h.end(); });

// unknown: the handshake only, no model call.
const unknown = randomUUID();
const c = await run("unknown", ["--resume", unknown], (h) => h.handshake(), (h) => h.end());

rec("driver", "summary", {
  newSession: id, resumedSessionIds: b.sessionIds, sameId: b.sessionIds.length > 0 && b.sessionIds.every((s) => s === id),
  resumedBeforeFirstUser: b.beforeFirstUser, answer: b.texts.join(" ").trim(), wordAnswered: b.texts.join(" ").includes(word),
  unknownId: unknown, unknownExit: c.code, unknownSessionIds: c.sessionIds, unknownResults: c.results,
});
process.exit(0);
