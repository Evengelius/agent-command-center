// Plan 38's probes P38x and P38t (plan.md, Plan 38): `codex app-server`
// with the dashboard's handshake, its initialize opting into Codex's
// experimental API (capabilities.experimentalApi, as OpenAI's Codex
// extension for VS Code does). P38x, no turn: thread/start as the
// dashboard sends it, then thread/settings/update to the effort in force
// (does a notification follow?), to "low" with a thread/read written as
// soon as the update is answered (what does it report?), to another model
// alone (is the effort kept?), and to P38t's model and effort, each
// followed by a wait for thread/settings/updated. P38t, one turn (the
// owner approved one or two live Codex turns on 2026-10-24): a message
// asking for a shell command that writes a file, so that a command
// approval comes, answered acceptForSession; currentTime/read answered
// with the time, as VS Code's extension answers it; then thread/read and,
// from the thread's rollout file, each turn_context's model and effort,
// nothing else of it. Records every line from the server except account
// and remote-control payloads (their method only); home, user and host
// names are replaced. Stops after the last step, or at 240 s.
// Usage: node p38-codex-settings.mjs <workdir> <outfile> [noturn]
import { spawn } from "node:child_process";
import { appendFileSync, readFileSync, writeFileSync } from "node:fs";

const [workdir, outfile, mode] = process.argv.slice(2);
const withTurn = mode !== "noturn";
const t0 = Date.now();
writeFileSync(outfile, "");

const esc = (x) => x.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
const home = process.env.USERPROFILE ?? "";
const hides = [
  [JSON.stringify(workdir).slice(1, -1), "<probe-work>"],
  [workdir, "<probe-work>"],
  [JSON.stringify(home).slice(1, -1), "<HOME>"], // as it reads inside a JSON string
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
const waiting = new Map(); // request id -> its resolve
const watchers = []; // {method, resolve}: the next notification of that method
const send = (obj) => { rec("in", obj); child.stdin.write(JSON.stringify(obj) + "\n"); };
const request = (method, params) => new Promise((resolve) => {
  const id = ++nextId;
  waiting.set(id, resolve);
  send(params === undefined ? { id, method } : { id, method, params });
});
// next resolves with the next notification of method, or null after ms.
const next = (method, ms) => new Promise((resolve) => {
  const w = { method, resolve };
  watchers.push(w);
  setTimeout(() => {
    const i = watchers.indexOf(w);
    if (i >= 0) { watchers.splice(i, 1); resolve(null); }
  }, ms);
});
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// The server's requests: the time as VS Code's extension gives it, a
// command approval for the session, a file change once; any other refused.
const reply = (m) => {
  switch (m.method) {
    case "currentTime/read":
      send({ id: m.id, result: { currentTimeAt: Math.floor(Date.now() / 1000) } });
      return;
    case "item/commandExecution/requestApproval":
      send({ id: m.id, result: { decision: "acceptForSession" } });
      return;
    case "item/fileChange/requestApproval":
      send({ id: m.id, result: { decision: "accept" } });
      return;
  }
  send({ id: m.id, error: { code: -32601, message: "probe: not handled" } });
};

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
      reply(m);
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
setTimeout(() => { rec("timeout", {}); child.kill(); }, 240000);

// The dashboard's own handshake (internal/provider/codex/session.go), with
// the opt-in.
const clientInfo = { name: "agent_command_center", title: "Agent Command Center", version: "dev" };
await request("initialize", { clientInfo, capabilities: { experimentalApi: true } });
send({ method: "initialized" });
const list = await request("model/list", {});
const models = (list.result?.data ?? []).filter((m) => !m.hidden);
const started = await request("thread/start", { sandbox: "read-only", approvalPolicy: "on-request", approvalsReviewer: "user" });
const thread = started.result?.thread?.id;
const model0 = started.result?.model;
const effort0 = started.result?.reasoningEffort;
rec("note", { step: "thread", model: model0, effort: effort0 });

// update writes one thread/settings/update and waits up to 1.5 s for the
// notification; then reads the thread.
const update = async (label, params, readAtOnce) => {
  rec("step", { label });
  const seen = next("thread/settings/updated", 1500);
  const answer = await request("thread/settings/update", { threadId: thread, ...params });
  const read = readAtOnce ? await request("thread/read", { threadId: thread }) : null;
  const n = await seen;
  rec("note", {
    step: label, answered: answer.error ? "error" : "ok", notified: !!n,
    notifiedModel: n?.params?.threadSettings?.model ?? null, notifiedEffort: n?.params?.threadSettings?.effort ?? null,
    readAtOnceEffort: read ? read.result?.thread?.reasoningEffort ?? null : undefined,
  });
  await sleep(300);
};

await update("the effort in force", { effort: effort0 });
await update("low, read at once", { effort: "low" }, true);
const efforts = (m) => (m?.supportedReasoningEfforts ?? []).map((e) => e.reasoningEffort);
const other = models.find((m) => m.model !== model0 && efforts(m).includes("low"));
await update("another model alone", { model: other?.model });
const cheap = models.find((m) => m.model !== model0 && /luna/i.test(m.model) && efforts(m).includes("low")) ?? other;
await update("the turn's model and effort", { model: cheap?.model, effort: "low" });
await request("thread/read", { threadId: thread });

if (withTurn) {
  rec("step", { label: "turn" });
  const done = next("turn/completed", 180000);
  await request("turn/start", {
    threadId: thread,
    input: [{ type: "text", text: "Use a shell command, not apply_patch, to create the file probe38.txt in the current directory containing exactly: ok. Then reply with the single word done." }],
    approvalsReviewer: "user", sandboxPolicy: { type: "readOnly" }, summary: "auto",
  });
  const end = await done;
  rec("note", { step: "turn", status: end?.params?.turn?.status ?? null });
  await request("thread/read", { threadId: thread });
  // The rollout's turn contexts, their model and effort alone.
  const path = started.result?.thread?.path;
  try {
    const contexts = [];
    for (const l of readFileSync(path, "utf8").split("\n")) {
      let o; try { o = JSON.parse(l); } catch { continue; }
      if (o.type === "turn_context") {
        contexts.push({ model: o.payload?.model ?? null, effort: o.payload?.effort ?? o.payload?.reasoning_effort ?? null });
      }
    }
    rec("rollout", { turnContexts: contexts });
  } catch (e) {
    rec("rollout", { error: String(e?.message ?? e).slice(0, 200) });
  }
}
child.stdin.end();
await exited;
rec("done", {});
process.exit(0);
