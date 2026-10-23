// Plan 34's probe P34c (doc/plan34-evaluation.md, Research): Claude Code with
// the dashboard's arguments (internal/provider/claude/session.go, args) and
// no user message, so no model call: the initialize response's models, then
// set_model, apply_flag_settings (effortLevel, ultracode) and get_settings
// after each, to see what each request answers and what the session reports
// in force. Records a filtered view only: the initialize response's models,
// version and plan, never its commands, agents or account names; from
// get_settings, `applied` and a fixed list of keys of `effective` and of each
// source, never the rest of the owner's settings. Home, user and host names
// are replaced in what is kept. The child's environment drops every CLAUDE*
// and ANTHROPIC_* variable: the owner's Windows environment sets none, and
// those of the session that runs the driver (CLAUDE_CODE_EFFORT_LEVEL among
// them) would change what the CLI applies. With "auto" (P34c2) or "empty"
// (P34c3) as a third argument, the process also gets --settings setting
// CLAUDE_CODE_EFFORT_LEVEL to "auto" or "" in its env, for this process only
// (no file written), and runs the second list of steps: the owner's
// settings set that variable, and "auto" still overrides every level, while
// "" leaves effort to the session, as without the variable. Stops after the
// last step, or at 150 s.
// Usage: node p34-claude-models.mjs <workdir> <outfile> [auto|empty]
import { spawn } from "node:child_process";
import { appendFileSync, writeFileSync } from "node:fs";

const [workdir, outfile, mode] = process.argv.slice(2);
const t0 = Date.now();
writeFileSync(outfile, "");

const esc = (x) => x.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
const home = process.env.USERPROFILE ?? "";
const hides = [
  [JSON.stringify(home).slice(1, -1), "<HOME>"], // as it reads inside a JSON string
  [home, "<HOME>"],
  [home.replaceAll("\\", "/"), "<HOME>"],
  [process.env.USERNAME ?? "", "<user>"],
  [process.env.COMPUTERNAME ?? "", "<host>"],
].filter(([s]) => s.length >= 3);
const clean = (s) => hides.reduce((a, [from, to]) => a.replace(new RegExp(esc(from), "gi"), to), s);
const rec = (dir, obj) =>
  appendFileSync(outfile, clean(JSON.stringify({ t_ms: Date.now() - t0, dir, msg: obj })) + "\n");

const KEYS = ["model", "effortLevel", "ultracode", "enableWorkflows", "disableWorkflows",
  "workflowKeywordTriggerEnabled", "availableModels", "maxEffortLevel", "modelSettings", "fastMode",
  "alwaysThinkingEnabled", "advisorModel"];
const pick = (o) => Object.fromEntries(KEYS.filter((k) => o && k in o).map((k) => [k, o[k]]));

const args = ["-p", "--input-format", "stream-json", "--output-format", "stream-json",
  "--verbose", "--include-partial-messages", "--forward-subagent-text",
  "--permission-prompt-tool", "stdio", "--permission-mode", "default"];
if (mode === "auto") args.push("--settings", JSON.stringify({ env: { CLAUDE_CODE_EFFORT_LEVEL: "auto" } }));
if (mode === "empty") args.push("--settings", JSON.stringify({ env: { CLAUDE_CODE_EFFORT_LEVEL: "" } }));
const env = Object.fromEntries(Object.entries(process.env).filter(([k]) => !/^(CLAUDE|ANTHROPIC_)/i.test(k)));
rec("meta", { argv: ["claude", ...args], cwd: "<probe-work>",
  dropped: Object.keys(process.env).filter((k) => !(k in env)).sort() });

const child = spawn("claude", args, { cwd: workdir, env, stdio: ["pipe", "pipe", "pipe"] });
let n = 0;
const sent = new Map(); // request_id to subtype
const send = (obj) => { child.stdin.write(JSON.stringify(obj) + "\n"); };

// The steps, each sent once the previous one is answered (or after 15 s).
const flags = (settings) => ({ subtype: "apply_flag_settings", settings });
const get = { subtype: "get_settings" };
const auto = [
  { subtype: "initialize", hooks: null }, get,
  flags({ effortLevel: "low" }), get,
  flags({ effortLevel: "max" }), get,
  flags({ ultracode: true }), get,
  flags({ effortLevel: "high", ultracode: true }), get,
  flags({ effortLevel: "medium" }), get,
  flags({ ultracode: false }), get,
  { subtype: "set_model", model: "sonnet" }, get,
  { subtype: "set_model", model: "claude-opus-4-6" }, flags({ effortLevel: "xhigh" }), get,
  { subtype: "set_model", model: "haiku" }, get,
  flags({ effortLevel: null }), get,
  { subtype: "set_model", model: "default" }, flags({ ultracode: null }), get,
];
const steps = mode === "auto" || mode === "empty" ? auto : [
  { subtype: "initialize", hooks: null },
  { subtype: "get_settings" },
  { subtype: "set_model", model: "sonnet" },
  { subtype: "get_settings" },
  { subtype: "apply_flag_settings", settings: { effortLevel: "low" } },
  { subtype: "get_settings" },
  { subtype: "apply_flag_settings", settings: { ultracode: true } },
  { subtype: "get_settings" },
  { subtype: "apply_flag_settings", settings: { effortLevel: "max" } },
  { subtype: "get_settings" },
  { subtype: "apply_flag_settings", settings: { ultracode: false } },
  { subtype: "get_settings" },
  { subtype: "set_model", model: "claude-opus-4-6" },
  { subtype: "apply_flag_settings", settings: { effortLevel: "xhigh" } },
  { subtype: "get_settings" },
  { subtype: "set_model", model: "haiku" },
  { subtype: "get_settings" },
  { subtype: "set_model", model: "fable" },
  { subtype: "get_settings" },
  { subtype: "set_model", model: "claude-not-a-model-34" },
  { subtype: "get_settings" },
  { subtype: "apply_flag_settings", settings: { effortLevel: "extreme" } },
  { subtype: "get_settings" },
  { subtype: "set_model", model: "default" },
  { subtype: "apply_flag_settings", settings: { effortLevel: null, ultracode: null } },
  { subtype: "get_settings" },
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
    return { models: r.models, claude_code_version: r.claude_code_version,
      account: { subscriptionType: r.account?.subscriptionType, apiProvider: r.account?.apiProvider },
      fast_mode_state: r.fast_mode_state, fast_mode_disabled_reason: r.fast_mode_disabled_reason,
      current_permission_mode: r.current_permission_mode, keys: Object.keys(r) };
  }
  if (subtype === "get_settings") {
    return { applied: r.applied, effective: pick(r.effective),
      sources: (r.sources ?? []).map((s) => ({ source: s.source, settings: pick(s.settings) })),
      errors: (r.errors ?? []).length, keys: Object.keys(r) };
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
      if (subtype) { sent.delete(id); next(); }
      continue;
    }
    if (m.type === "control_request") {
      rec("out", { type: m.type, request_id: m.request_id, subtype: m.request?.subtype });
      send({ type: "control_response", response: { subtype: "error", request_id: m.request_id, error: "probe: not handled" } });
      continue;
    }
    if (m.type === "system" && m.subtype === "informational") { rec("out", m); continue; }
    rec("out", { type: m.type, subtype: m.subtype, model: m.model });
  }
});
child.stderr.on("data", (d) => rec("stderr", d.toString("utf8").slice(0, 400)));
child.on("exit", (code, signal) => { rec("exit", { code, signal, steps: step }); process.exit(0); });
setTimeout(() => { rec("timeout", {}); child.kill(); }, 150000);
next();
