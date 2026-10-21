// Plan 22's live check P22s (plan.md, Plan 22, Proposed, not run; approved
// by the owner on 2026-10-21): Codex app-server with the dashboard's
// handshake and thread, as p12-codex.mjs (internal/provider/codex/session.go:
// sandbox read-only, approvals on request), and turn/start's summary, one
// value per turn from <summaries>, a comma list (TurnStartParams.summary:
// auto, concise, detailed, none). account/rateLimits/read at the start and at
// the end, for the quota used. Declines every approval and answers any other
// server request as not handled. Stops after the last turn, at 300 s, or on a
// refused turn/start, which is recorded and not retried. Records every line.
// Usage: node p22-codex.mjs <workdir> <outfile> <promptfile> <summaries>
import { spawn } from "node:child_process";
import { appendFileSync, writeFileSync, readFileSync } from "node:fs";

const [workdir, outfile, promptFile, summaryList] = process.argv.slice(2);
const prompts = readFileSync(promptFile, "utf8").split(/\r?\n/).map((s) => s.trim()).filter(Boolean);
const summaries = summaryList.split(",");
const t0 = Date.now();
writeFileSync(outfile, "");
const rec = (dir, obj) =>
  appendFileSync(outfile, JSON.stringify({ t_ms: Date.now() - t0, dir, msg: obj }) + "\n");
rec("argv", ["app-server"]);
const child = spawn("codex", ["app-server"], { cwd: workdir, stdio: ["pipe", "pipe", "pipe"], shell: true });
let nextId = 0;
const pending = new Map();
const send = (obj) => { rec("in", obj); child.stdin.write(JSON.stringify(obj) + "\n"); };
const request = (method, params, onResult, onError) => {
  const id = ++nextId; pending.set(id, { onResult, onError });
  send({ id, method, params });
};
let threadId, turn = 0, ending = false;
const end = () => { if (!ending) { ending = true; setTimeout(() => child.stdin.end(), 1000); } };
const limits = (then) => request("account/rateLimits/read", undefined, then, then);
const nextTurn = () => {
  if (turn >= prompts.length) { limits(end); return; }
  const summary = summaries[Math.min(turn, summaries.length - 1)];
  request("turn/start", { threadId, input: [{ type: "text", text: prompts[turn++] }], summary },
    null, () => limits(end)); // refused: recorded, not retried
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
      if (m.error) { rec("error", m.error); p?.onError?.(); }
      else if (p?.onResult) p.onResult(m.result);
    } else if (m.method === "turn/completed" && (m.params?.threadId ?? threadId) === threadId) nextTurn();
  }
});
child.stderr.on("data", (d) => rec("stderr", d.toString("utf8")));
child.on("exit", (code, signal) => { rec("exit", { code, signal, turns: turn }); process.exit(0); });
setTimeout(() => { rec("timeout", {}); child.kill(); }, 300000);
request("initialize", { clientInfo: { name: "acc-probe", title: "agent-command-center probe", version: "0.0.1" } }, () => {
  send({ method: "initialized" });
  limits(() => request("thread/start", { sandbox: "read-only", approvalPolicy: "on-request" },
    (r) => { threadId = r.thread.id; nextTurn(); }));
});
