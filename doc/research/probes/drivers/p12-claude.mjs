// Plan 12 probe: Claude Code with the dashboard's arguments
// (internal/provider/claude/session.go, args) and the permission mode given,
// one turn per prompt line; records every line; denies every can_use_tool.
// With <switch> = <mode>@<n>, sends set_permission_mode before turn n (1-based).
// Usage: node p12-claude.mjs <workdir> <outfile> <promptfile> <mode> [<switch>]
import { spawn } from "node:child_process";
import { appendFileSync, writeFileSync, readFileSync } from "node:fs";

const [workdir, outfile, promptFile, mode, sw] = process.argv.slice(2);
const prompts = readFileSync(promptFile, "utf8").split(/\r?\n/).map((s) => s.trim()).filter(Boolean);
const [swMode, swTurn] = sw ? [sw.split("@")[0], Number(sw.split("@")[1])] : [null, 0];
const t0 = Date.now();
writeFileSync(outfile, "");
const rec = (dir, obj) =>
  appendFileSync(outfile, JSON.stringify({ t_ms: Date.now() - t0, dir, msg: obj }) + "\n");

const args = ["-p", "--input-format", "stream-json", "--output-format", "stream-json",
  "--verbose", "--include-partial-messages", "--forward-subagent-text",
  "--permission-prompt-tool", "stdio", "--permission-mode", mode];
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

let turn = 0;
const next = () => {
  if (turn >= prompts.length) { child.stdin.end(); return; }
  const text = prompts[turn++];
  if (turn === swTurn) control({ subtype: "set_permission_mode", mode: swMode }, () => user(text));
  else user(text);
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
        response: { behavior: "deny", message: "probe: denied" } } });
    }
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
