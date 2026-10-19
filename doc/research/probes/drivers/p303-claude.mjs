// Re-check follow-up (2026-10-19): Claude Code with the dashboard's arguments
// (internal/provider/claude/session.go, args) and the live gate's two
// interrupts on a tool (internal/provider/claude/live_test.go, (d) and (c)),
// one turn per prompt; records every line. Turn 1 asks for a Write and is
// interrupted as soon as its can_use_tool arrives, which is never answered.
// Turn 2 asks for a ping, allows it and interrupts 3 s later, or 8 s after
// the Bash call when no approval is asked. Turn 3 checks the next turn runs;
// any other can_use_tool is denied.
// Usage: node p303-claude.mjs <workdir> <outfile>
import { spawn } from "node:child_process";
import { appendFileSync, writeFileSync } from "node:fs";

const [workdir, outfile] = process.argv.slice(2);
const prompts = [
  "Use the Write tool to create the file pending.txt in the current directory containing exactly: wait. Then reply with the single word done.",
  "Use the Bash tool to run exactly this command: ping -n 30 127.0.0.1. Then reply with the single word done.",
  "Reply with the single word ready.",
];
const t0 = Date.now();
writeFileSync(outfile, "");
const rec = (dir, obj) =>
  appendFileSync(outfile, JSON.stringify({ t_ms: Date.now() - t0, dir, msg: obj }) + "\n");

const args = ["-p", "--input-format", "stream-json", "--output-format", "stream-json",
  "--verbose", "--include-partial-messages", "--forward-subagent-text",
  "--permission-prompt-tool", "stdio", "--permission-mode", "default"];
rec("meta", { argv: ["claude", ...args], cwd: "<probe-work>" });

const child = spawn("claude", args, { cwd: workdir, stdio: ["pipe", "pipe", "pipe"] });
let n = 0;
const waiting = new Map();
const send = (obj) => { rec("in", obj); child.stdin.write(JSON.stringify(obj) + "\n"); };
const control = (request, then) => {
  const id = `req_${++n}`; if (then) waiting.set(id, then);
  send({ type: "control_request", request_id: id, request });
};
const user = (text) => send({ type: "user", message: { role: "user", content: text }, parent_tool_use_id: null, session_id: "default" });
const answer = (m, response) =>
  send({ type: "control_response", response: { subtype: "success", request_id: m.request_id, response } });

let turn = 0, timer = null, interrupted = false;
const interrupt = () => {
  if (interrupted) return;
  interrupted = true; clearTimeout(timer); timer = null;
  control({ subtype: "interrupt" });
};
const later = (ms) => { clearTimeout(timer); timer = setTimeout(interrupt, ms); };
const next = () => {
  interrupted = false; clearTimeout(timer); timer = null;
  if (turn >= prompts.length) { setTimeout(() => child.stdin.end(), 3000); return; }
  user(prompts[turn++]);
};

let buf = "";
child.stdout.on("data", (d) => {
  buf += d.toString("utf8");
  let i;
  while ((i = buf.indexOf("\n")) >= 0) {
    const line = buf.slice(0, i); buf = buf.slice(i + 1);
    if (!line.trim()) continue;
    let m; try { m = JSON.parse(line); } catch { rec("out-raw", line); continue; }
    rec("out", m);
    if (m.type === "control_request" && m.request?.subtype === "can_use_tool") {
      if (turn === 1) interrupt();
      else if (turn === 2 && m.request.tool_name === "Bash") { answer(m, { behavior: "allow", updatedInput: m.request.input }); later(3000); }
      else answer(m, { behavior: "deny", message: "probe: denied" });
    }
    if (turn === 2 && !interrupted && timer === null && m.type === "assistant"
        && (m.message?.content ?? []).some((c) => c.type === "tool_use" && c.name === "Bash")) later(8000);
    if (m.type === "control_response") {
      const then = waiting.get(m.response?.request_id);
      if (then) { waiting.delete(m.response.request_id); then(); }
    }
    if (m.type === "result") next();
  }
});
child.stderr.on("data", (d) => rec("stderr", d.toString("utf8")));
child.on("exit", (code, signal) => { rec("exit", { code, signal }); process.exit(0); });
setTimeout(() => { rec("timeout", {}); child.kill(); }, 300000);

control({ subtype: "initialize", hooks: null }, next);
