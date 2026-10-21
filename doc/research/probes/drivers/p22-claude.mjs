// Plan 22's live check P22o (plan.md, Plan 22, Proposed, not run; approved
// by the owner on 2026-10-21): Claude Code with the dashboard's arguments, as
// p303-claude.mjs (internal/provider/claude/session.go, args), the model set
// for this process only by ANTHROPIC_MODEL: Opus 5.5 where the standing
// decision says Haiku, which is the check's purpose. One prompt that needs
// reasoning and no tool; a second, asking it to think hard, only when the
// first turn had no thinking block. Every tool request is denied, any other
// request answered as not handled. Stops after the last result, or at 180 s.
// Records every line.
// Usage: ANTHROPIC_MODEL=claude-opus-5-5 node p22-claude.mjs <workdir> <outfile>
import { spawn } from "node:child_process";
import { appendFileSync, writeFileSync } from "node:fs";

const [workdir, outfile] = process.argv.slice(2);
const prompts = [
  "Without using any tool, find the smallest positive integer that leaves the remainders 1, 2, 3, 4 and 5 when divided by 2, 3, 4, 5 and 6. Think it through, then reply with the number and one line of justification.",
  "Think hard about this, without using any tool: in how many ways can 12 be written as an ordered sum of 1s, 2s and 3s? Reply with the number and one line of justification.",
];
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
const answer = (m, response) =>
  send({ type: "control_response", response: { subtype: "success", request_id: m.request_id, response } });

let turn = 0, thought = false, ending = false;
const next = () => {
  if (turn === 0 || (turn === 1 && !thought)) { user(prompts[turn++]); return; }
  if (!ending) { ending = true; setTimeout(() => child.stdin.end(), 2000); }
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
    if (m.type === "stream_event" && !m.parent_tool_use_id && m.event?.type === "content_block_start"
        && m.event.content_block?.type === "thinking") thought = true;
    if (m.type === "control_request") {
      if (m.request?.subtype === "can_use_tool") answer(m, { behavior: "deny", message: "probe: no tool in this check" });
      else send({ type: "control_response", response: { subtype: "error", request_id: m.request_id, error: "probe: not handled" } });
    }
    if (m.type === "control_response") {
      const then = waiting.get(m.response?.request_id);
      if (then) { waiting.delete(m.response.request_id); then(); }
    }
    if (m.type === "result") next();
  }
});
child.stderr.on("data", (d) => rec("stderr", d.toString("utf8")));
child.on("exit", (code, signal) => { rec("exit", { code, signal, turns: turn, thought }); process.exit(0); });
setTimeout(() => { rec("timeout", {}); child.kill(); }, 180000);

control({ subtype: "initialize", hooks: null }, next);
