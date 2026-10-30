// Plan 69's probe P69b (plan.md, Plan 69, criterion 69.1; run within the
// proposed quota as the owner's design review of 2026-10-29 asked): Codex
// resumed in a new app-server. The dashboard's handshake (initialize with its
// experimental API, initialized), thread parameters (sandbox read-only,
// approvals on request, reviewer user) and turn/start parameters
// (internal/provider/codex/session.go: Send, startThread). Four processes in
// turn:
//   new      thread/start, then one turn with a code word (one model call);
//   full     thread/resume on that thread with the dashboard's parameters as
//            they are: the response's size, its turns and their items, and
//            what follows it for 4 s (no model call);
//   exclude  thread/resume with excludeTurns true (0.163.0's experimental
//            schema: "Full-history hydration is deprecated for paginated
//            threads"), the same records, then one turn asking for the code
//            word (one model call);
//   unknown  thread/resume on a random UUID, then on "not-a-thread-69" (no
//            model call).
// Declines every approval and answers any other server request as not
// handled. A refused request is recorded and not retried. A process ends with
// stdin closed 1 s after its last step, or killed at 180 s. Records every
// line with the process's name; a thread/resume response as its size and a
// summary of its turns (ids, statuses, items view, item types), not the
// items themselves; home, user, host and work paths replaced.
// Usage: node p69-codex-resume.mjs <workdir> <outfile> <codeword>
import { spawn, execFileSync } from "node:child_process";
import { appendFileSync, writeFileSync } from "node:fs";
import { randomUUID } from "node:crypto";

const [workdir, outfile, word] = process.argv.slice(2);
if (!workdir || !outfile || !word) throw new Error("usage: node p69-codex-resume.mjs <workdir> <outfile> <codeword>");
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
  .replace(/[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[a-z]{2,}/g, "<email>")
  .replace(/RateLimitResetCredit_[0-9a-f]{8,}/g, "RateLimitResetCredit_<redacted>");
const rec = (proc, dir, obj) =>
  appendFileSync(outfile, clean(JSON.stringify({ t_ms: Date.now() - t0, proc, dir, msg: obj })) + "\n");
rec("driver", "meta", { cwd: "<probe-work>", word });

const clientInfo = { name: "agent_command_center", title: "Agent Command Center", version: "dev" };
const thread = { sandbox: "read-only", approvalPolicy: "on-request", approvalsReviewer: "user" };
const turnParams = (threadId, text) => ({ threadId, input: [{ type: "text", text }], approvalsReviewer: "user",
  approvalPolicy: "on-request", sandboxPolicy: { type: "readOnly" }, summary: "auto" });

// summary is a thread/resume (or thread/start) result without its items.
const summary = (r, bytes) => ({ bytes, keys: Object.keys(r ?? {}), model: r?.model, reasoningEffort: r?.reasoningEffort,
  approvalsReviewer: r?.approvalsReviewer, approvalPolicy: r?.approvalPolicy, sandbox: r?.sandbox,
  thread: r?.thread && { id: r.thread.id, sessionId: r.thread.sessionId, status: r.thread.status, preview: r.thread.preview,
    cwd: r.thread.cwd, cliVersion: r.thread.cliVersion, keys: Object.keys(r.thread),
    turns: (r.thread.turns ?? []).map((t) => ({ id: t.id, status: t.status, itemsView: t.itemsView,
      items: (t.items ?? []).map((it) => it.type) })) } });

// run starts one app-server and drives it with steps(h): each step is a
// request; next() moves on. Resolves with what was seen, once it exits.
const run = (proc, steps) => new Promise((resolve) => {
  rec(proc, "meta", { argv: ["codex", "app-server"] });
  const child = spawn("codex", ["app-server"], { cwd: workdir, stdio: ["pipe", "pipe", "pipe"] });
  const seen = { threadIds: [], errors: [], texts: [], tokenUsage: 0, afterResume: [] };
  let nextId = 0, resumedAt = 0;
  const pending = new Map();
  const send = (obj) => { rec(proc, "in", obj); child.stdin.write(JSON.stringify(obj) + "\n"); };
  const request = (method, params, then) => { const id = ++nextId; pending.set(id, { method, then }); send({ id, method, params }); };
  let ending = false;
  const end = () => { if (!ending) { ending = true; setTimeout(() => child.stdin.end(), 1000); } };
  const h = { request, end, seen, notify: (method) => send({ method }) };
  let onTurnDone = null;
  h.turn = (threadId, text, then) => { onTurnDone = then; request("turn/start", turnParams(threadId, text), (r, err) => { if (err) { onTurnDone = null; then(); } }); };
  let buf = "";
  child.stdout.on("data", (d) => {
    buf += d.toString("utf8");
    let i;
    while ((i = buf.indexOf("\n")) >= 0) {
      const line = buf.slice(0, i); buf = buf.slice(i + 1);
      if (!line.trim()) continue;
      let m; try { m = JSON.parse(line); } catch { rec(proc, "out-raw", line.slice(0, 2000)); continue; }
      if (resumedAt && Date.now() - resumedAt < 4000 && !(m.id !== undefined && !m.method)) seen.afterResume.push(m.method);
      if (m.id !== undefined && m.method) { // a server request
        rec(proc, "out", m);
        if (m.method.endsWith("/requestApproval")) send({ id: m.id, result: { decision: "decline" } });
        else send({ id: m.id, error: { code: -32601, message: "probe: not handled" } });
        continue;
      }
      if (m.id !== undefined) { // a response
        const p = pending.get(m.id); pending.delete(m.id);
        if (p && (p.method === "thread/resume" || p.method === "thread/start") && m.result) {
          rec(proc, "out", { id: m.id, result: summary(m.result, Buffer.byteLength(line)) });
          if (m.result.thread?.id) seen.threadIds.push(m.result.thread.id);
          if (p.method === "thread/resume") resumedAt = Date.now();
        } else rec(proc, "out", m);
        if (m.error) seen.errors.push({ method: p?.method, ...m.error });
        p?.then?.(m.result, m.error);
        continue;
      }
      rec(proc, "out", m);
      if (m.method === "thread/tokenUsage/updated") seen.tokenUsage++;
      if (m.method === "item/completed" && m.params?.item?.type === "agentMessage") seen.texts.push(m.params.item.text);
      if (m.method === "turn/completed" && onTurnDone) { const f = onTurnDone; onTurnDone = null; f(); }
    }
  });
  child.stderr.on("data", (d) => rec(proc, "stderr", d.toString("utf8").slice(0, 2000)));
  const killer = setTimeout(() => {
    rec(proc, "timeout", {});
    try { execFileSync("taskkill", ["/T", "/F", "/PID", String(child.pid)]); } catch { child.kill(); }
  }, 180000);
  child.on("exit", (code, signal) => { clearTimeout(killer); rec(proc, "exit", { code, signal, ...seen }); resolve({ code, ...seen }); });
  request("initialize", { clientInfo, capabilities: { experimentalApi: true } }, (r, err) => {
    if (err) { end(); return; }
    h.notify("initialized");
    steps(h);
  });
});

const ask = `Remember this code word for later in this conversation: ${word}. Reply with the single word OK, and nothing else.`;
const recall = "What is the code word I asked you to remember earlier in this conversation? Reply with the code word only.";
const after = (ms, f) => setTimeout(f, ms);

// new: a thread and the code word.
const a = await run("new", (h) => h.request("thread/start", thread, (r, err) => {
  if (err || !r?.thread?.id) { h.end(); return; }
  h.turn(r.thread.id, ask, () => h.end());
}));
const id = a.threadIds[0];
if (!id) { rec("driver", "stop", { reason: "the new thread reported no id" }); process.exit(0); }

// full: the dashboard's thread/resume as it is today.
const b = await run("full", (h) => h.request("thread/resume", { threadId: id, ...thread }, () => after(4000, () => h.end())));

// exclude: thread/resume without the turns, then the question.
const c = await run("exclude", (h) => h.request("thread/resume", { threadId: id, ...thread, excludeTurns: true }, (r, err) => {
  if (err || !r?.thread?.id) { h.end(); return; }
  after(4000, () => h.turn(r.thread.id, recall, () => h.end()));
}));

// unknown: two threads no rollout has.
const unknown = randomUUID();
const d = await run("unknown", (h) => h.request("thread/resume", { threadId: unknown, ...thread, excludeTurns: true },
  () => h.request("thread/resume", { threadId: "not-a-thread-69", ...thread, excludeTurns: true }, () => h.end())));

rec("driver", "summary", {
  newThread: id, fullThreadIds: b.threadIds, fullTokenUsage: b.tokenUsage, fullAfterResume: b.afterResume,
  excludeThreadIds: c.threadIds, excludeTokenUsage: c.tokenUsage, excludeAfterResume: c.afterResume,
  answer: c.texts.join(" ").trim(), wordAnswered: c.texts.join(" ").includes(word),
  unknownId: unknown, unknownErrors: d.errors, exits: [a.code, b.code, c.code, d.code],
});
process.exit(0);
