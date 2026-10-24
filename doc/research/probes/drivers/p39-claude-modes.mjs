// Plan 39's probe P39c (plan.md, Plan 39): Claude Code with the dashboard's
// arguments (internal/provider/claude/session.go, args) and no user message,
// so no model call: the initialize response's permission mode, then
// set_permission_mode with each mode the owner's selector names (default,
// acceptEdits, plan, auto), with one the CLI does not take, and back to
// default, to see what each request answers and what the session reports
// (system/status). Records a filtered view only: from the initialize
// response its version, its current_permission_mode and its keys, never its
// commands, agents or account names. Home, user and host names are replaced
// in what is kept. The child's environment drops every CLAUDE* and
// ANTHROPIC_* variable, as P34c's. Stops after the last step, or at 120 s.
// Usage: node p39-claude-modes.mjs <workdir> <outfile>
import { spawn } from "node:child_process";
import { appendFileSync, writeFileSync } from "node:fs";

const [workdir, outfile] = process.argv.slice(2);
const t0 = Date.now();
writeFileSync(outfile, "");

const esc = (x) => x.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
const home = process.env.USERPROFILE ?? "";
const hides = [
  [JSON.stringify(workdir).slice(1, -1), "<probe-work>"],
  [workdir, "<probe-work>"],
  [JSON.stringify(home).slice(1, -1), "<HOME>"],
  [home, "<HOME>"],
  [home.replaceAll("\\", "/"), "<HOME>"],
  [process.env.USERNAME ?? "", "<user>"],
  [process.env.COMPUTERNAME ?? "", "<host>"],
].filter(([s]) => s.length >= 3);
const clean = (s) => hides.reduce((a, [from, to]) => a.replace(new RegExp(esc(from), "gi"), to), s);
const rec = (dir, obj) =>
  appendFileSync(outfile, clean(JSON.stringify({ t_ms: Date.now() - t0, dir, msg: obj })) + "\n");

const args = ["-p", "--input-format", "stream-json", "--output-format", "stream-json",
  "--verbose", "--include-partial-messages", "--forward-subagent-text",
  "--permission-prompt-tool", "stdio", "--permission-mode", "default"];
const env = Object.fromEntries(Object.entries(process.env).filter(([k]) => !/^(CLAUDE|ANTHROPIC_)/i.test(k)));
rec("meta", { argv: ["claude", ...args], cwd: "<probe-work>",
  dropped: Object.keys(process.env).filter((k) => !(k in env)).sort() });

const child = spawn("claude", args, { cwd: workdir, env, stdio: ["pipe", "pipe", "pipe"] });
let n = 0;
const sent = new Map(); // request_id to subtype
const send = (obj) => { child.stdin.write(JSON.stringify(obj) + "\n"); };

const mode = (m) => ({ subtype: "set_permission_mode", mode: m });
const steps = [
  { subtype: "initialize", hooks: null },
  mode("default"), mode("acceptEdits"), mode("plan"), mode("auto"), mode("notAMode39"), mode("default"),
];
let step = 0, timer;
const next = () => {
  clearTimeout(timer);
  if (step >= steps.length) { setTimeout(() => child.stdin.end(), 1500); return; }
  const request = steps[step++];
  const id = `req_${++n}`;
  sent.set(id, request.subtype);
  rec("in", { type: "control_request", request_id: id, request });
  send({ type: "control_request", request_id: id, request });
  timer = setTimeout(() => { rec("step-timeout", { request_id: id }); next(); }, 15000);
};

const view = (subtype, r) => {
  if (!r || typeof r !== "object") return r;
  if (subtype === "initialize") {
    return { claude_code_version: r.claude_code_version, current_permission_mode: r.current_permission_mode,
      account: { subscriptionType: r.account?.subscriptionType, apiProvider: r.account?.apiProvider }, keys: Object.keys(r) };
  }
  return r;
};

let buf = "";
child.stdout.on("data", (d) => {
  buf += d.toString("utf8");
  let i;
  while ((i = buf.indexOf("\n")) >= 0) {
    const line = buf.slice(0, i); buf = buf.slice(i + 1);
    if (!line.trim()) continue;
    let m; try { m = JSON.parse(line); } catch { rec("out-raw", { length: line.length }); continue; }
    if (m.type === "control_response") {
      const id = m.response?.request_id;
      const subtype = sent.get(id);
      rec("out", { type: m.type, response: { subtype: m.response?.subtype, request_id: id, error: m.response?.error,
        response: view(subtype, m.response?.response) } });
      if (subtype) { sent.delete(id); setTimeout(next, 700); }
      continue;
    }
    if (m.type === "control_request") {
      rec("out", { type: m.type, request_id: m.request_id, subtype: m.request?.subtype });
      send({ type: "control_response", response: { subtype: "error", request_id: m.request_id, error: "probe: not handled" } });
      continue;
    }
    if (m.type === "system" && (m.subtype === "status" || m.subtype === "informational")) { rec("out", m); continue; }
    if (m.type === "system" && m.subtype === "init") { rec("out", { type: m.type, subtype: m.subtype, permissionMode: m.permissionMode, model: m.model }); continue; }
    rec("out", { type: m.type, subtype: m.subtype });
  }
});
child.stderr.on("data", (d) => rec("stderr", d.toString("utf8").slice(0, 400)));
child.on("exit", (code, signal) => { rec("exit", { code, signal, steps: step }); process.exit(0); });
setTimeout(() => { rec("timeout", {}); child.kill(); }, 120000);
next();
