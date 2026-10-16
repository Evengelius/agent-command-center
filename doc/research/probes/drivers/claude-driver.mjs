// Throwaway probe driver (plan 1, step 3). Drives `claude` over stream-json
// stdin/stdout with the Agent SDK's control protocol, records every line.
// Usage: node claude-driver.mjs <scenario> <workdir> <outfile>
import { spawn } from "node:child_process";
import { appendFileSync, writeFileSync } from "node:fs";

const [scenario, workdir, outfile] = process.argv.slice(2);
const t0 = Date.now();
writeFileSync(outfile, "");
const rec = (dir, obj) =>
  appendFileSync(outfile, JSON.stringify({ t_ms: Date.now() - t0, dir, msg: obj }) + "\n");

const args = [
  "--input-format", "stream-json", "--output-format", "stream-json", "--verbose",
  "--include-partial-messages", "--permission-prompt-tool", "stdio", "--model", "haiku",
];
if (scenario === "subagent") args.push("--forward-subagent-text", "--include-hook-events");
rec("meta", { argv: ["claude", ...args], cwd: "<probe-work>" });

const child = spawn("claude", args, { cwd: workdir, stdio: ["pipe", "pipe", "pipe"] });
let n = 0;
const send = (obj) => { rec("in", obj); child.stdin.write(JSON.stringify(obj) + "\n"); };
const control = (request) => send({ type: "control_request", request_id: `req_${++n}`, request });
const user = (text) => send({ type: "user", message: { role: "user", content: text }, parent_tool_use_id: null, session_id: "default" });

const prompts = {
  conversation: [
    "Create a file named hello.txt in the current directory containing the single word hello. Use the Write tool, then reply with the word done.",
    "Reply with the single word: again",
    "Count from 1 to 300, one number per line, nothing else.",
  ],
  subagent: [
    "Use the Agent tool to launch one general-purpose subagent with this task: list the files in the current directory, write their names into a new file named found.txt with the Write tool, and report the names. Then tell me in one line what it found.",
  ],
}[scenario];

let turn = 0, interrupted = false, results = 0, bg = 0;
const next = () => {
  if (turn < prompts.length) user(prompts[turn++]);
  else child.stdin.end();
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
      send({ type: "control_response", response: { subtype: "success", request_id: m.request_id,
        response: { behavior: "allow", updatedInput: m.request.input } } });
    }
    if (m.type === "control_response" && m.response?.request_id === "req_1") {
      control({ subtype: "get_context_usage" });
      next();
    }
    // Interrupt the long third turn as soon as it starts streaming text.
    if (scenario === "conversation" && turn === 3 && !interrupted && m.type === "stream_event"
        && m.event?.type === "content_block_delta") {
      interrupted = true;
      control({ subtype: "interrupt" });
    }
    // Keep stdin open while background subagents run: closing it aborts their
    // permission requests (first P3 run).
    if (m.type === "system" && m.subtype === "background_tasks_changed") bg = m.tasks.length;
    if (m.type === "result") { results++; if (turn < prompts.length || bg === 0) next(); }
  }
});
child.stderr.on("data", (d) => rec("stderr", d.toString("utf8")));
child.on("exit", (code, signal) => { rec("exit", { code, signal, results }); process.exit(0); });
setTimeout(() => { rec("timeout", {}); child.kill(); }, 240000);

control({ subtype: "initialize", hooks: null });
