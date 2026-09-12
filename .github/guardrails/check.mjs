// GENERATED FILE - DO NOT EDIT.
// Source: engineering/guardrails/api.md
// Regenerate: cd engineering && node tools/generate.mjs
//
// Self-contained on purpose: this repo's CI cannot check out the engineering
// repo, because actions/checkout of a second private repo needs a PAT rather
// than the default GITHUB_TOKEN. The rules are inlined below.

/**
 * Minimal glob to RegExp. Supports **, *, ? and {a,b} alternation.
 *
 * Two anchor modes, because guardrails match two different kinds of string:
 *   "full"    - a repo-relative file path, matched end to end (scope fields).
 *   "segment" - an import specifier, which may carry any prefix ("../../" or a
 *               Go module path). Anchored at a path-segment boundary so
 *               "components/**" hits "../../components/X" but not
 *               "src/subcomponents/X".
 */
export function globToRegExp(pattern, { anchor = "full" } = {}) {
  let out = "";
  for (let i = 0; i < pattern.length; i++) {
    const c = pattern[i];
    if (c === "*") {
      if (pattern[i + 1] === "*") {
        out += ".*";
        i++;
        if (pattern[i + 1] === "/") i++; // "**/" also matches zero segments
      } else {
        out += "[^/]*";
      }
    } else if (c === "?") {
      out += "[^/]";
    } else if (c === "{") {
      const close = pattern.indexOf("}", i);
      if (close === -1) throw new Error(`unclosed brace in glob: ${pattern}`);
      const alts = pattern.slice(i + 1, close).split(",");
      out += `(?:${alts.map(escapeLiteral).join("|")})`;
      i = close;
    } else {
      out += escapeLiteral(c);
    }
  }
  return anchor === "full"
    ? new RegExp(`^${out}$`)
    : new RegExp(`(?:^|/)${out}$`);
}

function escapeLiteral(s) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}






const GO_IMPORT = /^\s*(?:import\s+)?(?:[A-Za-z_.][\w.]*\s+)?"([^"]+)"\s*$/;
const TS_IMPORT = /(?:from|require\()\s*["']([^"']+)["']/;

/** Pulls import specifiers out of a source file, with their line numbers. */
export function extractImports(filePath, contents) {
  const isGo = filePath.endsWith(".go");
  const out = [];
  contents.split("\n").forEach((text, i) => {
    const m = isGo ? GO_IMPORT.exec(text) : TS_IMPORT.exec(text);
    if (m) out.push({ line: i + 1, spec: m[1] });
  });
  return out;
}

export function runRules(rules, { root, files, readFile } = {}) {
  const read =
    readFile ?? ((rel) => fs.readFileSync(path.join(root, rel), "utf8"));
  const violations = [];

  for (const rule of rules) {
    if (rule.kind === "existing-test") continue; // handled by existingTestViolations()
    const inScope = globToRegExp(rule.scope, { anchor: "full" });
    for (const file of files) {
      if (!inScope.test(file)) continue;
      let contents;
      try {
        contents = read(file);
      } catch (err) {
        if (err.code === "ENOENT") continue; // deleted in this diff
        throw err;
      }
      violations.push(...applyRule(rule, file, contents));
    }
  }
  return violations;
}

function applyRule(rule, file, contents) {
  switch (rule.kind) {
    case "forbidden-import":
      return checkImports(rule, file, contents);
    case "single-source-literal":
      return checkLiteral(rule, file, contents);
    case "deny-pattern":
      return checkPattern(rule, file, contents);
    default:
      throw new Error(`${rule.id}: engine has no handler for ${rule.kind}`);
  }
}

function checkImports(rule, file, contents) {
  if (matchesAny(rule.allow, file)) return [];
  const deny = rule.deny.map((g) => globToRegExp(g, { anchor: "segment" }));
  return extractImports(file, contents)
    .filter(({ spec }) => deny.some((re) => re.test(spec)))
    .filter(({ line }) => !allowed(rule.id, contents, line))
    .map(({ line, spec }) =>
      violation(rule, file, line, `imports "${spec}", which ${rule.scope} may not reach`)
    );
}

function checkLiteral(rule, file, contents) {
  if (file === rule.owner) return [];
  if (matchesAny(rule.allow_paths, file)) return [];
  const context = rule.context ? new RegExp(rule.context) : null;
  return scan(contents, (text) => {
    if (!text.includes(rule.literal)) return false;
    return context ? context.test(text) : true;
  })
    .filter(({ line }) => !allowed(rule.id, contents, line))
    .map(({ line }) =>
      violation(
        rule,
        file,
        line,
        `literal "${rule.literal}" belongs only in ${rule.owner}`
      )
    );
}

function checkPattern(rule, file, contents) {
  if (matchesAny(rule.allow_paths, file)) return [];
  const re = new RegExp(rule.pattern);
  return scan(contents, (text) => re.test(text))
    .filter(({ line }) => !allowed(rule.id, contents, line))
    .map(({ line }) => violation(rule, file, line, rule.message ?? `matches /${rule.pattern}/`));
}

function scan(contents, predicate) {
  const hits = [];
  contents.split("\n").forEach((text, i) => {
    if (predicate(text)) hits.push({ line: i + 1, text });
  });
  return hits;
}

function matchesAny(globs, file) {
  return (globs ?? []).some((g) => globToRegExp(g, { anchor: "full" }).test(file));
}

/** Escape rule ID for safe interpolation into RegExp. */
function escapeRegExp(s) {
  return s.replace(/[.*+?^${}()|[\]\\-]/g, "\\$&");
}

/**
 * Escape hatch, taken from uppsel: `guardrail-allow: <rule-id> - <reason>` in a
 * comment on the offending line. Deliberate, greppable, visible in review.
 * `guardrail-allow: *` suppresses every rule on that line.
 */
function allowed(ruleId, contents, line) {
  const text = contents.split("\n")[line - 1] ?? "";
  const escapedId = escapeRegExp(ruleId);
  return new RegExp(`guardrail-allow:\\s*(?:${escapedId}(?![\\w-])|\\*)`).test(text);
}

function violation(rule, file, line, message) {
  return { ruleId: rule.id, severity: rule.severity, file, line, message };
}

/**
 * An existing-test rule is a citation, not a check: it records a guardrail
 * enforced by a test that already exists. The one thing worth asserting is that
 * the test is still there, so deleting a guard fails loudly instead of quietly.
 */
export function existingTestViolations(rules, repoRoot) {
  return rules
    .filter((r) => r.kind === "existing-test")
    .filter((r) => !fs.existsSync(path.join(repoRoot, r.enforced_by)))
    .map((r) => ({
      ruleId: r.id,
      severity: "error",
      file: r.enforced_by,
      line: 1,
      message: `the test enforcing ${r.id} no longer exists`,
    }));
}

/**
 * Collect files from a repository. By default, walks the filesystem; if
 * fromDiff is set, returns only the files changed relative to that ref.
 * Uses git ls-files to respect .gitignore and exclude untracked directories.
 */
export function collectFiles(repoRoot, { fromDiff } = {}) {
  if (fromDiff) {
    try {
      const out = execFileSync(
        "git",
        ["diff", "--name-only", "--diff-filter=ACMR", `${fromDiff}...HEAD`],
        { cwd: repoRoot, encoding: "utf8" }
      );
      return out.split("\n").filter(Boolean);
    } catch (err) {
      throw new Error(`git diff failed with ref "${fromDiff}": ${err.message}`);
    }
  }
  try {
    const out = execFileSync("git", ["ls-files", "--cached", "--others", "--exclude-standard"], {
      cwd: repoRoot,
      encoding: "utf8",
    });
    return out.split("\n").filter(Boolean);
  } catch (err) {
    throw new Error(`git ls-files failed: ${err.message}`);
  }
}

/**
 * Format and print violations, returning the count of blocking errors.
 */
export function report(violations) {
  const ga = Boolean(process.env.GITHUB_ACTIONS);
  for (const v of violations) {
    if (ga) {
      process.stdout.write(
        `::${v.severity === "error" ? "error" : "warning"} file=${v.file},` +
          `line=${v.line},title=guardrail: ${v.ruleId}::${v.message}\n`
      );
    } else {
      process.stdout.write(
        `${v.severity === "error" ? "x" : "!"} ${v.file}:${v.line} [${v.ruleId}] ${v.message}\n`
      );
    }
  }
  const blocking = violations.filter((v) => v.severity === "error").length;
  process.stdout.write(
    blocking
      ? `\n${blocking} blocking violation(s). Fix them, or annotate the line with ` +
          `"guardrail-allow: <rule-id> - <reason>" if the exception is deliberate.\n`
      : `\nGuardrails clean (${violations.length} advisory).\n`
  );
  return blocking;
}

const RULES = [
  {
    "id": "GR-API-001",
    "kind": "forbidden-import",
    "scope": "internal/domain/**/*.go",
    "deny": [
      "internal/adapter/**",
      "internal/usecase/**",
      "internal/infra/**"
    ],
    "severity": "error",
    "mode": "repo"
  },
  {
    "id": "GR-API-002",
    "kind": "forbidden-import",
    "scope": "internal/usecase/**/*.go",
    "deny": [
      "internal/adapter/**",
      "internal/infra/**"
    ],
    "allow": [
      "internal/usecase/mocks/**"
    ],
    "severity": "error",
    "mode": "repo"
  },
  {
    "id": "GR-API-003",
    "kind": "forbidden-import",
    "scope": "internal/adapter/**/*.go",
    "deny": [
      "internal/infra/repo/**",
      "internal/infra/postgres/**"
    ],
    "severity": "error",
    "mode": "repo"
  },
  {
    "id": "GR-API-004",
    "kind": "forbidden-import",
    "scope": "internal/**/*.go",
    "deny": [
      "github.com/anthropics/**",
      "anthropic/**"
    ],
    "allow": [
      "internal/infra/claude/**"
    ],
    "severity": "error",
    "mode": "repo"
  },
  {
    "id": "GR-API-005",
    "kind": "deny-pattern",
    "scope": "internal/**/*.go",
    "pattern": "os\\.Getenv\\(",
    "allow_paths": [
      "internal/infra/config/**",
      "internal/**/*_adversarial_test.go"
    ],
    "message": "read configuration through internal/infra/config, not os.Getenv",
    "severity": "error",
    "mode": "repo"
  },
  {
    "id": "GR-API-006",
    "kind": "single-source-literal",
    "literal": "192",
    "owner": "internal/infra/claude/gateway.go",
    "scope": "internal/**/*.go",
    "context": "(Phone|phone|number|Number)\\s*[:=]",
    "allow_paths": [
      "internal/**/*_test.go"
    ],
    "severity": "error",
    "mode": "repo"
  },
  {
    "id": "GR-API-007",
    "kind": "deny-pattern",
    "scope": "internal/**/*.go",
    "pattern": "(fmt|log)\\.Print",
    "message": "use the zerolog logger, not fmt.Print or the stdlib log package",
    "severity": "error",
    "mode": "diff"
  },
  {
    "id": "GR-API-008",
    "kind": "forbidden-import",
    "scope": "internal/infra/**/*.go",
    "deny": [
      "internal/adapter/**"
    ],
    "allow": [
      "internal/infra/router/**"
    ],
    "severity": "error",
    "mode": "repo"
  },
  {
    "id": "GR-API-009",
    "kind": "deny-pattern",
    "scope": "internal/adapter/**/*.go",
    "pattern": "jackc/pgx|\\b(SELECT|INSERT INTO|DELETE FROM)\\b|UPDATE .* SET",
    "message": "handlers reach the database through a usecase and a repository, never directly",
    "severity": "error",
    "mode": "diff"
  }
];

// collectFiles, existingTestViolations and report come from the engine
// concatenated above. This runner is argv parsing and the exit call, nothing
// else, so the file-collection strategy cannot drift between the local CLI and
// the committed per-repo checkers.
//
// fs, path and execFileSync look unused here: they are used by the inlined
// engine, whose own import lines strip() removed.
import * as fs from "node:fs";
import * as path from "node:path";
import { execFileSync } from "node:child_process";

const ROOT = process.cwd();

const argv = process.argv.slice(2);
const i = argv.indexOf("--from-diff");
if (i !== -1 && (argv[i + 1] === undefined || argv[i + 1].startsWith("--"))) {
  process.stderr.write("usage: check.mjs [--from-diff <base-ref>]\n");
  process.exit(2);
}
const fromDiff = i === -1 ? null : argv[i + 1];

const all = collectFiles(ROOT);
const changed = fromDiff ? collectFiles(ROOT, { fromDiff }) : all;

const violations = [
  ...runRules(RULES.filter((r) => r.mode === "repo"), { root: ROOT, files: all }),
  ...runRules(RULES.filter((r) => r.mode === "diff"), { root: ROOT, files: changed }),
  ...existingTestViolations(RULES, ROOT),
];

process.exit(report(violations) ? 1 : 0);
