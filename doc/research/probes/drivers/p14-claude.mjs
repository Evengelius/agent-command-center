// Plan 14 probe: Claude Code with the dashboard's arguments
// (internal/provider/claude/session.go, args), one turn per prompt line;
// records every line. An AskUserQuestion request is allowed with
// updatedInput {questions, answers} (doc agent-sdk/user-input): the option
// whose label holds "café" for a single-select question, else its first
// option; the first two options joined by ", " for a multiSelect one. Any
// other can_use_tool is denied. After each result, get_context_usage is
// sent (CC-10 q10), and the next turn starts on its response.
// Usage: node p14-claude.mjs <workdir> <outfile> <promptfile>
import { spawn } from "node:child_process";
import { appendFileSync, writeFileSync, readFileSync } from "node:fs";

const [workdir, outfile, promptFile] = process.argv.slice(2);
const prompts = readFileSync(promptFile, "utf8").split(/\r?\n/).map((s) => s.trim()).filter(Boolean);
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

const answer = (input) => {
  const answers = {};
  for (const q of input.questions ?? []) {
    const labels = (q.options ?? []).map((o) => o.label);
    answers[q.question] = q.multiSelect ? labels.slice(0, 2).join(", ")
      : (labels.find((l) => l.includes("café")) ?? labels[0]);
  }
  return { questions: input.questions, answers };
};

let turn = 0;
const next = () => {
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
      const response = m.request.tool_name === "AskUserQuestion"
        ? { behavior: "allow", updatedInput: answer(m.request.input) }
        : { behavior: "deny", message: "probe: denied" };
      send({ type: "control_response", response: { subtype: "success", request_id: m.request_id, response } });
    }
    if (m.type === "control_response") {
      const then = waiting.get(m.response?.request_id);
      if (then) { waiting.delete(m.response.request_id); then(); }
    }
    if (m.type === "result") control({ subtype: "get_context_usage" }, next);
  }
});
child.stderr.on("data", (d) => rec("stderr", d.toString("utf8")));
child.on("exit", (code, signal) => { rec("exit", { code, signal }); process.exit(0); });
setTimeout(() => { rec("timeout", {}); child.kill(); }, 300000);

control({ subtype: "initialize", hooks: null }, next);
