// Plan 34's probe P34x (doc/plan34-evaluation.md, Research): `codex
// app-server` with the dashboard's handshake (initialize with clientInfo
// only) and no turn, so no model call. Run 1: model/list one model a page
// following nextCursor, then the default page and the one with hidden
// models; thread/start as the dashboard sends it, thread/read, then
// thread/settings/update (model and effort) and thread/read again. Run 2,
// a new process with capabilities.experimentalApi: the same update, a
// reasoning effort the model lists last, an effort it does not list, an
// unknown model, each followed by thread/read. Records every line from the
// server except account and remote-control payloads (their method only);
// home, user and host names are replaced. Stops after run 2, or at 120 s.
// Usage: node p34-codex-models.mjs <workdir> <outfile>
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

// run starts one app-server and runs its steps in order: each step is a
// function of the results so far that returns [method, params] (a request,
// answered before the next step), a number (wait that many ms), or null
// (skip). The process ends after the last step.
const run = (label, capabilities, steps) => new Promise((done) => {
  rec("run", { label, capabilities: capabilities ?? null });
  const child = spawn("codex", ["app-server"], { cwd: workdir, stdio: ["pipe", "pipe", "pipe"], shell: true });
  let nextId = 0, i = 0;
  const results = {};
  const pending = new Map();
  const send = (obj) => { rec("in", obj); child.stdin.write(JSON.stringify(obj) + "\n"); };
  const advance = () => {
    if (i >= steps.length) { child.stdin.end(); return; }
    const s = steps[i++](results);
    if (s === null) { advance(); return; }
    if (typeof s === "number") { setTimeout(advance, s); return; }
    const [method, params, name] = s;
    const id = ++nextId;
    pending.set(id, name ?? method);
    send(params === undefined ? { id, method } : { id, method, params });
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
        send({ id: m.id, error: { code: -32601, message: "probe: not handled" } });
      } else if (m.id !== undefined && pending.has(m.id)) {
        const name = pending.get(m.id); pending.delete(m.id);
        results[name] = m.result ?? { error: m.error };
        if (name === "initialize") send({ method: "initialized" });
        advance();
      }
    }
  });
  child.stderr.on("data", (d) => rec("stderr", d.toString("utf8").slice(0, 400)));
  child.on("exit", (code, signal) => { rec("exit", { label, code, signal }); done(); });
  setTimeout(() => { rec("timeout", { label }); child.kill(); }, 60000);
  advance();
});

const clientInfo = { name: "acc-probe", title: "agent-command-center probe", version: "0.0.1" };
const thread = { cwd: workdir, sandbox: "read-only", approvalPolicy: "on-request" };
const visible = (r) => (r["model/list"]?.data ?? []).filter((m) => !m.hidden);
const other = (r) => {
  const now = r["thread/start"]?.model;
  return visible(r).find((m) => m.model !== now) ?? visible(r)[0];
};
const id = (r) => r["thread/start"]?.thread?.id;

// Run 1: the stable surface, as the dashboard connects.
const page = (n) => (r) => {
  const prev = r[`page ${n - 1}`];
  if (n > 1 && !prev?.nextCursor) return null;
  return ["model/list", n === 1 ? { limit: 1 } : { limit: 1, cursor: prev.nextCursor }, `page ${n}`];
};
await run("stable", null, [
  () => ["initialize", { clientInfo }],
  ...Array.from({ length: 12 }, (_, n) => page(n + 1)),
  () => ["model/list", {}],
  () => ["model/list", { includeHidden: true }, "model/list hidden"],
  () => ["thread/start", thread],
  (r) => ["thread/read", { threadId: id(r) }, "read 1"],
  (r) => ["thread/settings/update", { threadId: id(r), model: other(r)?.model, effort: "low" }],
  () => 1500,
  (r) => ["thread/read", { threadId: id(r) }, "read 2"],
]);

// Run 2: with the experimental opt-in.
await run("experimental", { experimentalApi: true }, [
  () => ["initialize", { clientInfo, capabilities: { experimentalApi: true } }],
  () => ["model/list", {}],
  () => ["thread/start", thread],
  (r) => ["thread/settings/update", { threadId: id(r), model: other(r)?.model, effort: "low" }, "update model"],
  () => 1500,
  (r) => ["thread/read", { threadId: id(r) }, "read 1"],
  (r) => {
    const m = other(r);
    const last = m?.supportedReasoningEfforts?.at(-1)?.reasoningEffort;
    return last ? ["thread/settings/update", { threadId: id(r), effort: last }, "update last effort"] : null;
  },
  () => 1500,
  (r) => ["thread/read", { threadId: id(r) }, "read 2"],
  (r) => ["thread/settings/update", { threadId: id(r), effort: "extreme" }, "update unknown effort"],
  () => 1500,
  (r) => ["thread/read", { threadId: id(r) }, "read 3"],
  (r) => ["thread/settings/update", { threadId: id(r), model: "gpt-not-a-model-34" }, "update unknown model"],
  () => 1500,
  (r) => ["thread/read", { threadId: id(r) }, "read 4"],
]);
rec("done", {});
process.exit(0);
