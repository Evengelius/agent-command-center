// Plan 24's probe P24u (doc/plan24-evaluation.md, "After the owner's live
// session"; approved by the owner on 2026-10-22): Claude Code with the
// dashboard's arguments, as p22-claude.mjs (internal/provider/claude/
// session.go, args), the model set for this process only by ANTHROPIC_MODEL
// (Haiku, the standing decision), one message: "/usage", to see what the
// CLI answers over stream-json (claude-code.md, section 7: the usage report
// and its rows; CC-10 q9). Every tool request is denied, any other request
// answered as not handled. Stops 3 s after the first result, or at 90 s.
// Records every line.
// Usage: ANTHROPIC_MODEL=haiku node p24-claude-usage.mjs <workdir> <outfile>
import { spawn } from "node:child_process";
import { appendFileSync, writeFileSync } from "node:fs";

const [workdir, outfile] = process.argv.slice(2);
const t0 = Date.now();
writeFileSync(outfile, "");
const rec = (dir, obj) =>
  appendFileSync(outfile, JSON.stringify({ t_ms: Date.now() - t0, dir, msg: obj }) + "\n");

const args = ["-p", "--input-format", "stream-json", "--output-format", "stream-json",
  "--verbose", "--include-partial-messages", "--forward-subagent-text",
  "--permission-prompt-tool", "stdio", "--permission-mode", "default"];
rec("meta", { argv: ["claude", ...args], cwd: "<probe-work>", ANTHROPIC_MODEL: process.env.ANTHROPIC_MODEL ?? null });

const child = spawn("claude", args, { cwd: workdir, stdio: ["pipe", "pipe", "pipe"] });
let n = 0;
const waiting = new Map();
const send = (obj) => { rec("in", obj); child.stdin.write(JSON.stringify(obj) + "\n"); };
const control = (request, then) => {
  const id = `req_${++n}`; if (then) waiting.set(id, then);
  send({ type: "control_request", request_id: id, request });
};
const user = (text) => send({ type: "user", message: { role: "user", content: text }, parent_tool_use_id: null, session_id: "default" });

let sent = false, ending = false;
const end = () => { if (!ending) { ending = true; setTimeout(() => child.stdin.end(), 3000); } };

let buf = "";
child.stdout.on("data", (d) => {
  buf += d.toString("utf8");
  let i;
  while ((i = buf.indexOf("\n")) >= 0) {
    const line = buf.slice(0, i); buf = buf.slice(i + 1);
    if (!line.trim()) continue;
    let m; try { m = JSON.parse(line); } catch { rec("out-raw", line); continue; }
    rec("out", m);
    if (m.type === "control_request") {
      if (m.request?.subtype === "can_use_tool") send({ type: "control_response", response: { subtype: "success", request_id: m.request_id, response: { behavior: "deny", message: "probe: no tool in this check" } } });
      else send({ type: "control_response", response: { subtype: "error", request_id: m.request_id, error: "probe: not handled" } });
    }
    if (m.type === "control_response") {
      const then = waiting.get(m.response?.request_id);
      if (then) { waiting.delete(m.response.request_id); then(); }
    }
    if (m.type === "result") end();
  }
});
child.stderr.on("data", (d) => rec("stderr", d.toString("utf8")));
child.on("exit", (code, signal) => { rec("exit", { code, signal, sent }); process.exit(0); });
setTimeout(() => { rec("timeout", {}); child.kill(); }, 90000);

control({ subtype: "initialize", hooks: null }, () => { sent = true; user("/usage"); });
