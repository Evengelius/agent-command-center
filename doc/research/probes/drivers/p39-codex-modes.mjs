// Plan 39's probe P39x (plan.md, Plan 39): `codex app-server` with the
// dashboard's handshake, its initialize opting into the experimental API
// (plan 38), and no turn, so no model call. configRequirements/read (the
// managed requirements: which approval policies and sandboxes are allowed);
// thread/start as the dashboard sends it (read-only, on-request, the user
// reviewing); then thread/settings/update with each preset of OpenAI's Codex
// extension for VS Code (openai.chatgpt 26.1021.22027): "Ask for approval"
// (workspace-write, on-request, user), "Approve for me" (workspace-write,
// on-request, auto_review), "Full access" (danger-full-access, never, user),
// then back to read-only, and read-only again (a change of nothing), each
// followed by a wait for thread/settings/updated. The thread runs no turn:
// nothing runs under any of these settings. Records every line from the
// server except account and remote-control payloads (their method only);
// home, user and host names are replaced. Stops after the last step, or at
// 120 s.
// Usage: node p39-codex-modes.mjs <workdir> <outfile>
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
const quiet = (method) => method.startsWith("account/") || method.startsWith("remoteControl/");

const child = spawn("codex", ["app-server"], { cwd: workdir, stdio: ["pipe", "pipe", "pipe"], shell: true });
let nextId = 0;
const waiting = new Map();
const watchers = [];
const send = (obj) => { rec("in", obj); child.stdin.write(JSON.stringify(obj) + "\n"); };
const request = (method, params) => new Promise((resolve) => {
  const id = ++nextId;
  waiting.set(id, resolve);
  send(params === undefined ? { id, method } : { id, method, params });
});
const next = (method, ms) => new Promise((resolve) => {
  const w = { method, resolve };
  watchers.push(w);
  setTimeout(() => {
    const i = watchers.indexOf(w);
    if (i >= 0) { watchers.splice(i, 1); resolve(null); }
  }, ms);
});

let buf = "";
child.stdout.on("data", (d) => {
  buf += d.toString("utf8");
  let k;
  while ((k = buf.indexOf("\n")) >= 0) {
    const line = buf.slice(0, k); buf = buf.slice(k + 1);
    if (!line.trim()) continue;
    let m; try { m = JSON.parse(line); } catch { rec("out-raw", { length: line.length }); continue; }
    if (m.method && quiet(m.method)) { rec("out", { method: m.method }); continue; }
    rec("out", m);
    if (m.id !== undefined && m.method) {
      send({ id: m.id, error: { code: -32601, message: "probe: not handled" } });
    } else if (m.id !== undefined && waiting.has(m.id)) {
      const resolve = waiting.get(m.id); waiting.delete(m.id);
      resolve(m);
    } else if (m.method) {
      for (const w of [...watchers]) {
        if (w.method === m.method) { watchers.splice(watchers.indexOf(w), 1); w.resolve(m); }
      }
    }
  }
});
child.stderr.on("data", (d) => rec("stderr", d.toString("utf8").slice(0, 400)));
const exited = new Promise((r) => child.on("exit", (code, signal) => { rec("exit", { code, signal }); r(); }));
setTimeout(() => { rec("timeout", {}); child.kill(); }, 120000);

const clientInfo = { name: "agent_command_center", title: "Agent Command Center", version: "dev" };
await request("initialize", { clientInfo, capabilities: { experimentalApi: true } });
send({ method: "initialized" });
const req = await request("configRequirements/read");
rec("note", { step: "requirements", requirements: req.result?.requirements ?? null, error: req.error ?? null });
const started = await request("thread/start", { sandbox: "read-only", approvalPolicy: "on-request", approvalsReviewer: "user" });
const thread = started.result?.thread?.id;
rec("note", { step: "thread", sandbox: started.result?.sandbox, approvalPolicy: started.result?.approvalPolicy,
  approvalsReviewer: started.result?.approvalsReviewer, activePermissionProfile: started.result?.activePermissionProfile });

const update = async (label, params) => {
  rec("step", { label });
  const seen = next("thread/settings/updated", 1500);
  const answer = await request("thread/settings/update", { threadId: thread, ...params });
  const n = await seen;
  const s = n?.params?.threadSettings;
  rec("note", { step: label, answered: answer.error ? answer.error : "ok", notified: !!n,
    approvalPolicy: s?.approvalPolicy ?? null, approvalsReviewer: s?.approvalsReviewer ?? null,
    sandboxPolicy: s?.sandboxPolicy ?? null, activePermissionProfile: s?.activePermissionProfile ?? null });
  await new Promise((r) => setTimeout(r, 300));
};

await update("Ask for approval", { sandboxPolicy: { type: "workspaceWrite" }, approvalPolicy: "on-request", approvalsReviewer: "user" });
await update("Approve for me", { sandboxPolicy: { type: "workspaceWrite" }, approvalPolicy: "on-request", approvalsReviewer: "auto_review" });
await update("Full access", { sandboxPolicy: { type: "dangerFullAccess" }, approvalPolicy: "never", approvalsReviewer: "user" });
await update("read-only again", { sandboxPolicy: { type: "readOnly" }, approvalPolicy: "on-request", approvalsReviewer: "user" });
await update("read-only, no change", { sandboxPolicy: { type: "readOnly" }, approvalPolicy: "on-request", approvalsReviewer: "user" });
child.stdin.end();
await exited;
rec("done", {});
process.exit(0);
