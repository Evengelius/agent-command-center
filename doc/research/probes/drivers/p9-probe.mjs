// Plan 9 probe: Codex 0.163.0 with per-process -c overrides, the dashboard's
// handshake (initialize with clientInfo only, thread/start without cwd),
// one turn per prompt line; records every line; answers each approval with
// the decision given (default accept). Usage:
// node p9-probe.mjs <workdir> <outfile> <promptfile> <decision> [-c k=v ...]
import { spawn } from "node:child_process";
import { appendFileSync, writeFileSync, readFileSync } from "node:fs";

const [workdir, outfile, promptFile, decision = "accept", ...extra] = process.argv.slice(2);
const prompts = readFileSync(promptFile, "utf8").split(/\r?\n/).map((s) => s.trim()).filter(Boolean);
const t0 = Date.now();
writeFileSync(outfile, "");
const rec = (dir, obj) =>
  appendFileSync(outfile, JSON.stringify({ t_ms: Date.now() - t0, dir, msg: obj }) + "\n");
rec("argv", [...extra, "app-server"]);
const child = spawn("codex", [...extra, "app-server"], { cwd: workdir, stdio: ["pipe", "pipe", "pipe"], shell: true });
let nextId = 0;
const pending = new Map();
const send = (obj) => { rec("in", obj); child.stdin.write(JSON.stringify(obj) + "\n"); };
const request = (method, params, onResult) => {
  const id = ++nextId; pending.set(id, { onResult });
  send(params === undefined ? { id, method } : { id, method, params });
};
let threadId, turn = 0;
const nextTurn = () => {
  if (turn >= prompts.length) { child.stdin.end(); return; }
  request("turn/start", { threadId, input: [{ type: "text", text: prompts[turn++] }] });
};
const onServerRequest = (m) => {
  if (m.method.endsWith("/requestApproval")) send({ id: m.id, result: { decision } });
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
setTimeout(() => { rec("timeout", {}); child.kill(); }, 240000);
request("initialize", { clientInfo: { name: "acc-probe", title: "agent-command-center probe", version: "0.0.1" } }, () => {
  send({ method: "initialized" });
  request("thread/start", {}, (r) => { threadId = r.thread.id; nextTurn(); });
});
