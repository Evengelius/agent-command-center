// Plan 76's probe P76a (plan.md, Plan 76, design D8; criterion 76.15; run
// within the plan's approved usage): Claude Code running a project's own
// agent definitions, the Implementer and the Reviewer, as no-tool canaries.
// The dashboard's arguments (internal/provider/claude/session.go, args) and
// handshake (initialize, then get_settings and the user message "/usage",
// which Claude Code answers locally, with no model call: P24u), with
// CLAUDE_CODE_EMIT_SESSION_STATE_EVENTS as the dashboard sets it; every
// CLAUDE* and ANTHROPIC_* variable of this shell dropped, and Haiku as the
// orchestrator's per-process model (ANTHROPIC_MODEL=haiku, the standing
// decision). The work folder holds copies of this repository's
// .claude/agents/implementer.md and reviewer.md, made before the run.
// One process, one turn: the orchestrator runs each agent once with "Use no
// tool. Reply READY.", then, if it has a SendMessage tool, continues the
// implementer once. Tool requests for Agent, Task and SendMessage are allowed,
// every other tool request denied, any other request answered as not handled.
// The process ends with stdin closed 1.5 s after the turn's result, or is
// killed with its whole tree at 300 s, its limit (doc/evaluation.md,
// Execution supervision: L75's driver had 6 minutes for a turn of the same
// size; this turn runs two short canaries). Records every line; the
// initialize and get_settings responses and system/init as filtered views
// (of the account, its subscription type and API provider alone; no
// commands, skills, plugins or MCP servers; the agents' names and whether
// SendMessage is offered kept); every other line as the CLI wrote it, the
// "/usage" answer and system/commands_changed included; home, user, host and
// work paths replaced.
// Usage: node p76-claude-agents.mjs <workdir> <outfile>
import { spawn, execFileSync } from "node:child_process";
import { appendFileSync, writeFileSync } from "node:fs";

const [workdir, outfile] = process.argv.slice(2);
if (!workdir || !outfile) throw new Error("usage: node p76-claude-agents.mjs <workdir> <outfile>");
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
rec("driver", "meta", { cwd: "<probe-work>", limit_s: 300,
  dropped: Object.keys(process.env).filter((k) => !(k in env)).sort(),
  set: { CLAUDE_CODE_EMIT_SESSION_STATE_EVENTS: "1", ANTHROPIC_MODEL: "haiku" } });

const view = (subtype, r) => {
  if (!r || typeof r !== "object") return r;
  if (subtype === "initialize") {
    return { claude_code_version: r.claude_code_version, current_permission_mode: r.current_permission_mode,
      account: { subscriptionType: r.account?.subscriptionType, apiProvider: r.account?.apiProvider },
      agents: Array.isArray(r.agents) ? r.agents.map((a) => a?.name ?? a) : r.agents,
      session: Object.fromEntries(Object.entries(r).filter(([k]) => /session/i.test(k))), keys: Object.keys(r) };
  }
  if (subtype === "get_settings") return { applied: r.applied, keys: Object.keys(r) };
  return r;
};
const toolNames = (t) => (Array.isArray(t) ? t.map((x) => (typeof x === "string" ? x : x?.name)) : []);
const initView = (m) => ({ type: m.type, subtype: m.subtype, session_id: m.session_id, model: m.model, cwd: m.cwd,
  permissionMode: m.permissionMode, claude_code_version: m.claude_code_version, apiKeySource: m.apiKeySource,
  tools: Array.isArray(m.tools) ? m.tools.length : m.tools,
  agentTools: toolNames(m.tools).filter((n) => /^(Agent|Task|SendMessage)$/.test(n)),
  agents: Array.isArray(m.agents) ? m.agents : undefined, keys: Object.keys(m) });

const allowed = new Set(["Agent", "Task", "SendMessage"]);
const prompt = "Use the Agent tool to run the implementer agent, then the reviewer agent, each with this prompt: Use no tool. Reply READY. If you have a SendMessage tool, then send the implementer agent: Reply READY again. Finally reply DONE.";

const seen = { sessionIds: [], results: 0, texts: [], tasks: [], agentModels: {}, toolUses: [], denied: [] };
const argv = [...base];
rec("p76a", "meta", { argv: ["claude", ...argv] });
const child = spawn("claude", argv, { cwd: workdir, env, stdio: ["pipe", "pipe", "pipe"] });
let n = 0;
const sent = new Map();
const send = (obj) => { rec("p76a", "in", obj); child.stdin.write(JSON.stringify(obj) + "\n"); };
const control = (request) => { const id = `req_${++n}`; sent.set(id, request.subtype); send({ type: "control_request", request_id: id, request }); };
const user = (text) => send({ type: "user", message: { role: "user", content: text }, parent_tool_use_id: null, session_id: "default" });
let ending = false;
const end = () => { if (!ending) { ending = true; setTimeout(() => child.stdin.end(), 1500); } };
let buf = "";
child.stdout.on("data", (d) => {
  buf += d.toString("utf8");
  let i;
  while ((i = buf.indexOf("\n")) >= 0) {
    const line = buf.slice(0, i); buf = buf.slice(i + 1);
    if (!line.trim()) continue;
    let m; try { m = JSON.parse(line); } catch { rec("p76a", "out-raw", line.slice(0, 2000)); continue; }
    if (m.session_id && m.session_id !== "default" && !seen.sessionIds.includes(m.session_id)) seen.sessionIds.push(m.session_id);
    if (m.type === "control_response") {
      const id = m.response?.request_id, subtype = sent.get(id);
      sent.delete(id);
      rec("p76a", "out", { type: m.type, response: { subtype: m.response?.subtype, request_id: id, error: m.response?.error,
        response: view(subtype, m.response?.response) } });
      if (subtype === "initialize") { control({ subtype: "get_settings" }); user("/usage"); }
      continue;
    }
    if (m.type === "system" && m.subtype === "init") { rec("p76a", "out", initView(m)); continue; }
    rec("p76a", "out", m);
    if (m.type === "system" && /^task_/.test(m.subtype ?? "")) seen.tasks.push({ subtype: m.subtype, task_id: m.task_id, subagent_type: m.subagent_type, status: m.status ?? m.patch?.status });
    if (m.type === "control_request") {
      const tool = m.request?.tool_name;
      if (m.request?.subtype === "can_use_tool" && allowed.has(tool)) send({ type: "control_response", response: { subtype: "success", request_id: m.request_id, response: { behavior: "allow", updatedInput: m.request?.input ?? {} } } });
      else if (m.request?.subtype === "can_use_tool") { seen.denied.push(tool); send({ type: "control_response", response: { subtype: "success", request_id: m.request_id, response: { behavior: "deny", message: "probe: only the Agent, Task and SendMessage tools in this check" } } }); }
      else send({ type: "control_response", response: { subtype: "error", request_id: m.request_id, error: "probe: not handled" } });
    }
    if (m.type === "assistant") {
      if (m.parent_tool_use_id && m.message?.model) seen.agentModels[m.parent_tool_use_id] = m.message.model;
      for (const c of m.message?.content ?? []) {
        if (c.type === "tool_use") seen.toolUses.push({ name: c.name, agent: c.input?.subagent_type ?? c.input?.to ?? undefined, by: m.parent_tool_use_id ?? "orchestrator" });
        if (c.type === "text" && !m.parent_tool_use_id) seen.texts.push(c.text);
      }
    }
    if (m.type === "result") { seen.results++; if (seen.results === 1) user(prompt); else end(); }
  }
});
child.stderr.on("data", (d) => rec("p76a", "stderr", d.toString("utf8").slice(0, 2000)));
const killer = setTimeout(() => {
  rec("p76a", "timeout", { limit_s: 300 });
  try { execFileSync("taskkill", ["/T", "/F", "/PID", String(child.pid)]); } catch { child.kill(); }
}, 300000);
child.on("exit", (code, signal) => {
  clearTimeout(killer);
  rec("p76a", "exit", { code, signal });
  rec("driver", "summary", { sessionIds: seen.sessionIds, results: seen.results, tasks: seen.tasks,
    agentModels: seen.agentModels, toolUses: seen.toolUses, denied: seen.denied, answer: seen.texts.join(" ").trim(),
    sendMessageUsed: seen.toolUses.some((u) => u.name === "SendMessage") });
  process.exit(0);
});
control({ subtype: "initialize", hooks: null });
