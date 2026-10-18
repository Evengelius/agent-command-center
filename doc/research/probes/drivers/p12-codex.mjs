// Plan 12 probe: Codex app-server with the dashboard's handshake and thread
// parameters (internal/provider/codex/session.go: sandbox read-only,
// approvals on-request) plus approvalsReviewer, one turn per prompt line;
// records every line; declines every approval that reaches it. <reviewers>
// is a comma list, one per turn ("-" sends none): the first goes on
// thread/start, each on its turn/start. <sandboxes>, optional, is a comma
// list the same way for turn/start's sandboxPolicy type (readOnly,
// workspaceWrite; "-" sends none). <threadSandbox>, optional, is thread/start's
// sandbox mode (default read-only).
// Usage: node p12-codex.mjs <workdir> <outfile> <promptfile> <reviewers> [<sandboxes> [<threadSandbox>]]
import { spawn } from "node:child_process";
import { appendFileSync, writeFileSync, readFileSync } from "node:fs";

const [workdir, outfile, promptFile, reviewerList, sandboxList = "-", threadSandbox = "read-only"] =
  process.argv.slice(2);
const prompts = readFileSync(promptFile, "utf8").split(/\r?\n/).map((s) => s.trim()).filter(Boolean);
const reviewers = reviewerList.split(",").map((r) => (r === "-" ? undefined : r));
const sandboxes = sandboxList.split(",").map((r) => (r === "-" ? undefined : r));
const t0 = Date.now();
writeFileSync(outfile, "");
const rec = (dir, obj) =>
  appendFileSync(outfile, JSON.stringify({ t_ms: Date.now() - t0, dir, msg: obj }) + "\n");
rec("argv", ["app-server"]);
const child = spawn("codex", ["app-server"], { cwd: workdir, stdio: ["pipe", "pipe", "pipe"], shell: true });
let nextId = 0;
const pending = new Map();
const send = (obj) => { rec("in", obj); child.stdin.write(JSON.stringify(obj) + "\n"); };
const request = (method, params, onResult) => {
  const id = ++nextId; pending.set(id, { onResult });
  send({ id, method, params });
};
const withReviewer = (p, r) => (r ? { ...p, approvalsReviewer: r } : p);
const withSandbox = (p, t) => (t ? { ...p, sandboxPolicy: { type: t } } : p);
let threadId, turn = 0;
const nextTurn = () => {
  if (turn >= prompts.length) { child.stdin.end(); return; }
  const r = reviewers[Math.min(turn, reviewers.length - 1)];
  const sb = sandboxes[Math.min(turn, sandboxes.length - 1)];
  request("turn/start", withSandbox(withReviewer({ threadId, input: [{ type: "text", text: prompts[turn++] }] }, r), sb));
};
const onServerRequest = (m) => {
  if (m.method.endsWith("/requestApproval")) send({ id: m.id, result: { decision: "decline" } });
  else send({ id: m.id, error: { code: -32601, message: "probe: not handled" } });
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
    if (m.id !== undefined && m.method) onServerRequest(m);
    else if (m.id !== undefined) {
      const p = pending.get(m.id); pending.delete(m.id);
      if (p?.onResult && m.result) p.onResult(m.result);
      if (m.error) rec("error", m.error);
    } else if (m.method === "turn/completed" && (m.params?.threadId ?? threadId) === threadId) nextTurn();
  }
});
child.stderr.on("data", (d) => rec("stderr", d.toString("utf8")));
child.on("exit", (code, signal) => { rec("exit", { code, signal }); process.exit(0); });
setTimeout(() => { rec("timeout", {}); child.kill(); }, 300000);
request("initialize", { clientInfo: { name: "acc-probe", title: "agent-command-center probe", version: "0.0.1" } }, () => {
  send({ method: "initialized" });
  request("thread/start", withReviewer({ sandbox: threadSandbox, approvalPolicy: "on-request" }, reviewers[0]),
    (r) => { threadId = r.thread.id; nextTurn(); });
});
