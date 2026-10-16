// Throwaway probe driver (plan 1, step 3, P6). Drives `codex app-server` over
// stdio JSON-RPC (schema from `codex app-server generate-json-schema`),
// records every line. Usage: node codex-driver.mjs <workdir> <outfile> [model] [prompt]
// With a prompt (P7): one turn with that prompt, no interrupt turn; only the
// parent thread's turn/completed ends it (subagent threads report their own).
import { spawn } from "node:child_process";
import { appendFileSync, writeFileSync } from "node:fs";

const [workdir, outfile, model, prompt] = process.argv.slice(2);
const t0 = Date.now();
writeFileSync(outfile, "");
const rec = (dir, obj) =>
  appendFileSync(outfile, JSON.stringify({ t_ms: Date.now() - t0, dir, msg: obj }) + "\n");

const args = ["app-server"];
rec("meta", { argv: ["codex", ...args], cwd: "<probe-work>" });
const child = spawn("codex", args, { cwd: workdir, stdio: ["pipe", "pipe", "pipe"] });

let nextId = 0;
const pending = new Map();
const send = (obj) => { rec("in", obj); child.stdin.write(JSON.stringify(obj) + "\n"); };
const request = (method, params, onResult) => {
  const id = ++nextId; pending.set(id, { method, onResult });
  send(params === undefined ? { id, method } : { id, method, params });
};
const notify = (method) => send({ method });

let threadId, turnNo = 0, turnId, interrupted = false;
const startTurn = (text) => {
  turnNo++;
  request("turn/start", { threadId, input: [{ type: "text", text }] }, (r) => { turnId = r.turn.id; });
};

const onNotification = (m) => {
  if (prompt && m.params?.threadId && m.params.threadId !== threadId) return;
  if (m.method === "turn/started") turnId = m.params?.turn?.id ?? turnId;
  // Interrupt the long second turn as soon as the answer starts streaming.
  if (turnNo === 2 && !interrupted && m.method === "item/agentMessage/delta") {
    interrupted = true;
    request("turn/interrupt", { threadId, turnId });
  }
  if (m.method === "turn/completed") {
    if (turnNo === 1 && !prompt) startTurn("Count from 1 to 300, one number per line, nothing else.");
    else request("account/rateLimits/read", undefined, () => child.stdin.end());
  }
};

const onServerRequest = (m) => {
  // Approve what the agent asks for: the folder is a throwaway probe folder.
  let result;
  if (m.method === "item/commandExecution/requestApproval" || m.method === "item/fileChange/requestApproval")
    result = { decision: "accept" };
  else if (m.method === "applyPatchApproval" || m.method === "execCommandApproval")
    result = { decision: "approved" };
  if (result) send({ id: m.id, result });
  else send({ id: m.id, error: { code: -32601, message: "probe driver: not handled" } });
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
    } else if (m.method) onNotification(m);
  }
});
child.stderr.on("data", (d) => rec("stderr", d.toString("utf8")));
child.on("exit", (code, signal) => { rec("exit", { code, signal, turns: turnNo }); process.exit(0); });
setTimeout(() => { rec("timeout", {}); child.kill(); }, 240000);

request("initialize", { clientInfo: { name: "acc-probe", title: "agent-command-center probe", version: "0.0.1" } }, () => {
  notify("initialized");
  request("account/rateLimits/read", undefined, () => {
    request("thread/start", { cwd: workdir, sandbox: "read-only", approvalPolicy: "on-request", ...(model ? { model } : {}) }, (r) => {
      threadId = r.thread.id;
      if (prompt) return startTurn(prompt);
      startTurn("Create a file named hello.txt in the current directory containing the single word hello, then reply with the word done.");
    });
  });
});
