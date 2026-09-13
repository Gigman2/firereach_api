// GENERATED FILE - DO NOT EDIT.
// Source: engineering/tools/comment.mjs
// Regenerate: cd engineering && node tools/generate.mjs

/**
 * Posts a pull request's CI results as one comment in its conversation, and
 * keeps that single comment current on every push.
 *
 * Each workflow step saves its output to a file; this script is then called
 * with every step's outcome and log, renders a markdown summary, and creates
 * or edits the comment through the `gh` CLI that GitHub's runners already
 * carry, using the job's own token. A hidden marker is how a later run finds
 * the comment again, so a busy PR gets one report rather than one per push.
 *
 * Generated into each repo's .github/guardrails/ from engineering/tools/,
 * like the checker, so the app and api repos cannot drift apart.
 */
import * as fs from "node:fs";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

export const MARKER = "<!-- firereach-ci-report -->";

// GitHub rejects comment bodies over 65,536 characters.
const MAX_BODY = 65536;
const MAX_VIOLATIONS = 50;
const LOG_TAIL_LINES = 40;
const KINDS = ["setup", "tsc", "jest", "go", "gobuild"];

// Jest colours its output; the escape codes are noise in markdown.
const ANSI = /\x1b\[[0-9;]*[A-Za-z]/g;
const clean = (text) => String(text ?? "").replace(ANSI, "");
const plural = (n, word) => `${n} ${word}${n === 1 ? "" : "s"}`;

export function summarizeJest(log) {
  const lines = clean(log).split("\n");
  const summary = lines.find((l) => /^Tests:\s/.test(l));
  if (!summary) return null;
  const count = (word) => Number(summary.match(new RegExp(`(\\d+) ${word}`))?.[1] ?? 0);
  // Jest marks each failure with a bullet, lists it twice (in place and again
  // in the closing summary), and uses the same bullet for console output.
  const failing = [];
  for (const l of lines) {
    const name = l.match(/^\s*● (.+)$/)?.[1].trim();
    if (name && name !== "Console" && !failing.includes(name)) failing.push(name);
  }
  return { passed: count("passed"), failed: count("failed"), skipped: count("skipped"), failing };
}

export function summarizeGoTest(log) {
  const lines = clean(log).split("\n");
  const failing = [];
  for (const l of lines) {
    const name = l.match(/^\s*--- FAIL: (\S+)/)?.[1];
    if (name && !failing.includes(name)) failing.push(name);
  }
  return {
    ok: lines.filter((l) => /^ok\s/.test(l)).length,
    // "FAIL<tab><package>" per failing package; a bare "FAIL" is Go's run summary.
    failed: lines.filter((l) => /^FAIL\s+\S/.test(l)).length,
    failing,
  };
}

export function summarizeTsc(log) {
  return { errors: clean(log).split("\n").filter((l) => /error TS\d+:/.test(l)).length };
}

// "skipped", "cancelled", and "" (a step that never ran) all mean the check
// said nothing about the code.
function stateOf(outcome) {
  if (outcome === "success") return "passed";
  if (outcome === "failure") return "failed";
  return "not-run";
}

function guardrailsRow({ outcome, report }) {
  if (!report) {
    // A failure with no report means the checker never got as far as writing
    // one: a usage error or a git failure, not a verdict about the code.
    const state = stateOf(outcome);
    const result = state === "failed" ? "❌ checker error" : state === "passed" ? "✅ passed" : "⏭ skipped";
    return { name: "Guardrails", result, state };
  }
  if (report.blocking) {
    return { name: "Guardrails", result: `❌ ${plural(report.blocking, "blocking violation")}`, state: "failed" };
  }
  const advisory = report.advisory ? `, ${report.advisory} advisory` : "";
  return { name: "Guardrails", result: `✅ clean (${report.rules} rules)${advisory}`, state: "passed" };
}

function stepDetail({ kind, log }, ok) {
  switch (kind) {
    case "tsc": {
      const { errors } = summarizeTsc(log);
      if (ok) return "tsc clean";
      return errors ? plural(errors, "type error") : "failed";
    }
    case "gobuild":
      return ok ? "build clean" : "build failed";
    case "jest": {
      const s = summarizeJest(log);
      if (!s) return ok ? "passed" : "failed";
      return [s.failed && `${s.failed} failed`, `${s.passed} passed`, s.skipped && `${s.skipped} skipped`]
        .filter(Boolean)
        .join(", ");
    }
    case "go": {
      const s = summarizeGoTest(log);
      if (s.failed) return `${plural(s.failed, "package")} failed, ${s.ok} passed`;
      return ok ? `${plural(s.ok, "package")} passed` : "failed";
    }
    default:
      return ok ? "done" : "failed";
  }
}

function stepRow(step) {
  const state = stateOf(step.outcome);
  if (state === "not-run") return { name: step.name, result: "⏭ skipped", state };
  const ok = state === "passed";
  return { name: step.name, result: `${ok ? "✅" : "❌"} ${stepDetail(step, ok)}`, state };
}

function cell(value) {
  return String(value).replace(/\|/g, "\\|").replace(/\r?\n/g, " ");
}

function originOf(url) {
  try {
    return new URL(url).origin;
  } catch {
    return "https://github.com";
  }
}

function violationsTable(violations, repo, sha, origin) {
  const lines = ["| Rule | Where | What |", "|---|---|---|"];
  for (const v of violations.slice(0, MAX_VIOLATIONS)) {
    const where = `[${cell(`${v.file}:${v.line}`)}](${origin}/${repo}/blob/${sha}/${v.file}#L${v.line})`;
    const what = cell(v.message) + (v.severity === "error" ? "" : " _(advisory)_");
    lines.push(`| ${cell(v.ruleId)} | ${where} | ${what} |`);
  }
  const rest = violations.length - MAX_VIOLATIONS;
  if (rest > 0) lines.push("", `…and ${rest} more. The run's log has the full list.`);
  return lines.join("\n");
}

function longestBacktickRun(text) {
  return Math.max(0, ...(text.match(/`+/g) ?? []).map((run) => run.length));
}

// The last lines of a log in a collapsible block. The fence is always longer
// than any run of backticks in the log, so the log cannot close it early.
function outputBlock(title, log, failing = []) {
  const tail = clean(log).trimEnd().split("\n").slice(-LOG_TAIL_LINES).join("\n");
  const fence = "`".repeat(Math.max(3, longestBacktickRun(tail) + 1));
  const lines = [`<details><summary>${title}</summary>`, ""];
  if (failing.length) lines.push("Failing:", ...failing.map((name) => `- ${name}`), "");
  lines.push(`${fence}text`, tail, fence, "", "</details>");
  return lines.join("\n");
}

export function buildComment({ repo, sha, runUrl, guardrails, steps }) {
  const sha7 = String(sha ?? "").slice(0, 7) || "unknown";
  const rows = [
    guardrailsRow(guardrails),
    // Installing dependencies earns a row only when it failed: then it is
    // why the checks after it were skipped.
    ...steps.filter((s) => !(s.kind === "setup" && s.outcome === "success")).map(stepRow),
  ];
  const failed = rows.filter((r) => r.state === "failed").length;
  const headline = failed
    ? `❌ ${plural(failed, "check")} failed`
    : rows.some((r) => r.state === "not-run")
      ? "⚠️ not every check ran"
      : "✅ all checks passed";

  const parts = [
    MARKER,
    `### CI ${headline} · \`${sha7}\``,
    "",
    "| Check | Result |",
    "|---|---|",
    ...rows.map((r) => `| ${r.name} | ${r.result} |`),
    "",
  ];

  const violations = guardrails.report?.violations ?? [];
  if (violations.length) parts.push(violationsTable(violations, repo, sha, originOf(runUrl)), "");
  if (!guardrails.report && stateOf(guardrails.outcome) === "failed") {
    parts.push(outputBlock("Guardrails output", guardrails.log), "");
  }
  for (const step of steps) {
    if (step.outcome !== "failure") continue;
    const failing =
      step.kind === "jest" ? (summarizeJest(step.log)?.failing ?? [])
      : step.kind === "go" ? summarizeGoTest(step.log).failing
      : [];
    parts.push(outputBlock(`${step.name} output`, step.log, failing), "");
  }
  if (guardrails.report && !violations.length) {
    parts.push(
      "<details><summary>Guardrail report</summary>",
      "",
      `No violations across ${plural(guardrails.report.rules, "rule")}. ` +
        "Ratchet rules checked only the lines this PR changed.",
      "",
      "</details>",
      ""
    );
  }
  parts.push(`<sub>[View the run](${runUrl}) · this comment is updated on every push</sub>`);

  const body = parts.join("\n");
  return body.length <= MAX_BODY
    ? body
    : body.slice(0, MAX_BODY - 120) + "\n\n…trimmed to fit GitHub's comment size limit. The run has everything.";
}

// The bot's own report, found by its marker. A person quoting the marker in a
// comment of their own must never have that comment overwritten.
export function findReportComment(comments) {
  return comments.find((c) => c.type === "Bot" && String(c.body ?? "").includes(MARKER)) ?? null;
}

function runGh(args, input) {
  return execFileSync("gh", args, { encoding: "utf8", input, stdio: ["pipe", "pipe", "inherit"] });
}

export function upsertComment({ repo, pr, body, gh = runGh }) {
  const listed = gh([
    "api", "--paginate", `repos/${repo}/issues/${pr}/comments`,
    "--jq", ".[] | {id: .id, type: .user.type, body: .body}",
  ])
    .split("\n")
    .filter(Boolean)
    .map((line) => JSON.parse(line));
  const existing = findReportComment(listed);
  const input = JSON.stringify({ body });
  if (existing) {
    gh(["api", "--method", "PATCH", `repos/${repo}/issues/comments/${existing.id}`, "--input", "-"], input);
    return "updated";
  }
  gh(["api", "--method", "POST", `repos/${repo}/issues/${pr}/comments`, "--input", "-"], input);
  return "created";
}

/**
 * --guardrails "outcome|report.json|log"           required, exactly once
 * --step "Name|outcome|kind|log"                   repeatable, kind in KINDS
 * --dry-run                                        print instead of posting
 * An outcome may be empty: that is what a step that never ran reports.
 */
export function parseCommentArgs(argv) {
  let guardrails = null;
  const steps = [];
  let dryRun = false;
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === "--dry-run") {
      dryRun = true;
      continue;
    }
    if (arg !== "--guardrails" && arg !== "--step") return null;
    const spec = argv[++i];
    if (spec === undefined) return null;
    const parts = spec.split("|");
    if (arg === "--guardrails") {
      if (parts.length !== 3) return null;
      const [outcome, json, log] = parts;
      guardrails = { outcome, json, log };
    } else {
      if (parts.length !== 4) return null;
      const [name, outcome, kind, log] = parts;
      if (!name || !KINDS.includes(kind)) return null;
      steps.push({ name, outcome, kind, log });
    }
  }
  return guardrails ? { guardrails, steps, dryRun } : null;
}

// A step that was skipped never wrote its log; that is not an error here.
function readOrEmpty(file) {
  try {
    return fs.readFileSync(file, "utf8");
  } catch {
    return "";
  }
}

function main() {
  const args = parseCommentArgs(process.argv.slice(2));
  if (!args) {
    process.stderr.write(
      'usage: comment.mjs --guardrails "outcome|report.json|log" [--step "Name|outcome|kind|log"]... [--dry-run]\n'
    );
    process.exit(2);
  }
  const { GITHUB_REPOSITORY: repo, PR_NUMBER: pr, HEAD_SHA, GITHUB_SHA } = process.env;
  const server = process.env.GITHUB_SERVER_URL ?? "https://github.com";
  let report = null;
  try {
    report = JSON.parse(fs.readFileSync(args.guardrails.json, "utf8"));
  } catch {
    report = null;
  }
  const body = buildComment({
    repo,
    sha: HEAD_SHA || GITHUB_SHA || "",
    runUrl: `${server}/${repo}/actions/runs/${process.env.GITHUB_RUN_ID}`,
    guardrails: { outcome: args.guardrails.outcome, report, log: readOrEmpty(args.guardrails.log) },
    steps: args.steps.map((s) => ({ ...s, log: readOrEmpty(s.log) })),
  });
  if (args.dryRun) {
    process.stdout.write(body + "\n");
    return;
  }
  if (!repo || !pr) {
    process.stderr.write("comment.mjs: GITHUB_REPOSITORY and PR_NUMBER are required to post\n");
    process.exit(2);
  }
  const result = upsertComment({ repo, pr, body });
  process.stdout.write(`CI report comment ${result} on ${repo}#${pr}\n`);
}

if (process.argv[1] === fileURLToPath(import.meta.url)) main();
