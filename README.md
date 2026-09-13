# agentlint

[![CI](https://github.com/quietmachineworks/agentlint/actions/workflows/ci.yml/badge.svg)](https://github.com/quietmachineworks/agentlint/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/quietmachineworks/agentlint.svg)](https://pkg.go.dev/github.com/quietmachineworks/agentlint)
[![Go Report Card](https://goreportcard.com/badge/github.com/quietmachineworks/agentlint)](https://goreportcard.com/report/github.com/quietmachineworks/agentlint)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Resolve what an agent's configuration declares against what is actually on the
machine. Settings, hooks, subagents, skills. No model, no network, no tokens.

```bash
go install github.com/quietmachineworks/agentlint/cmd/agentlint@latest
agentlint
```

Or take a binary for your platform from
[Releases](https://github.com/quietmachineworks/agentlint/releases), which carry
Linux, macOS and Windows on both architectures.

Or from a clone, with nothing installed but Go:

```bash
git clone https://github.com/quietmachineworks/agentlint
cd agentlint && go build ./cmd/agentlint
```

```
/Users/you/.claude
2 settings, 22 hook commands, 34 subagents, 876 skills

settings.json
  error    permisions: "permisions" is not a setting; "permissions" differs by one edit
           settings-unknown-key
           rename "permisions" to "permissions"

1 error(s), 4 warning(s)
```

## Usage

```
agentlint [--config DIR] [--json] [--strict] [--version]
```

`--config` reads a directory other than `CLAUDE_CONFIG_DIR` or `~/.claude`.
`--json` writes the run for another program, counts included. `--strict` fails
on warnings too.

Exit 0 clean, 1 with errors, 2 when the configuration directory cannot be read.
Nothing is written, ever, whatever the flags.

### In CI

A configuration lives in a repository as often as on a laptop, and it rots the
same way. The exit code is the whole integration:

```yaml
- name: The agent configuration this repository ships still resolves
  run: |
    go run github.com/quietmachineworks/agentlint/cmd/agentlint@latest \
      --config .claude --strict
```

`--strict` is the right setting for CI and the wrong one for a laptop: an
undocumented settings key is a warning worth seeing once, not a broken build
every morning.

## Why it exists

A published JSON schema already describes `settings.json`, and
`claude plugin validate --strict` already judges plugin manifests. Neither
answers the only question that matters once a config has been alive for a year:
**is any of this still true?**

A schema says a hook entry is well formed. It cannot say the script it names was
deleted two laptops ago. That gap is the whole tool.

Measured against the runtime validator, these pass in silence today: a skill
whose `name` disagrees with its folder, a skill with no `name` at all, a
`description` over budget, a frontmatter key that is a typo, a hook whose
command does not resolve, and every subagent definition, which
`claude plugin validate` does not discover at all.

## Checks

- `settings-schema` - the whole file against the published schema, which is
  embedded. A pattern mismatch is a warning rather than an error: a regex in a
  published schema is the likeliest place for it to be a simplification of the
  parser it describes.
- `hook-unresolvable` - a hook names a program that is missing, is a directory,
  or is not executable. A command whose target only the runtime can expand is
  reported as unverifiable, never as broken.
- `settings-unknown-key` - the schema sets `additionalProperties: true`, so a
  misspelt top-level key is silent. One edit away from a real key is an error
  with the correction; anything further is a warning, because undocumented keys
  are real.
- `settings-unknown-hook-event` - a hook keyed on an event that will never fire.
- `agent-frontmatter` - subagent definitions: name, agreement with the filename,
  description, budget.
- `skill-frontmatter` - the same for skills, including the ones plugins bring,
  which are carried on every prompt exactly like the agent's own.
- `mcp-unresolvable` - a server whose command is missing or not on PATH, an http
  server with no url, an entry with nothing to start. A server that cannot start
  costs a connection attempt every session and says nothing when it fails.
- `secret-in-config` - a credential written into a settings or MCP file, named
  by field and shape and never by value.
- `permission-rule` - an allowance that grants a whole tool, a rule another
  already covers, and a rule a denial has already taken back. The runtime
  reaches deny before ask and ask before allow, so a rule can sit in the file
  deciding nothing.
- `settings-precedence` - a scalar set at two levels where only the higher one
  is read. Keys the runtime merges rather than replaces are left alone, and a
  file whose place in the stack is not published never decides a claim.
- `settings-scope` - a key sitting in a file the runtime reads, under a name the
  runtime reads, at a level that does not honour it. Managed-only keys in a
  project file are the common case, and nothing warns about them.
- `name-collision` - one name installed twice. The runtime reaches one of them,
  and the copy that loses looks dormant while being a duplicate.
- `description-shadowing` - two descriptions that are the same text under
  different names. The floor is high on purpose: two hardening skills, one for
  Linux and one for Windows, share most of their vocabulary and none of their
  purpose, and reporting those is how a linter gets muted.

## What it does not do

It does not judge whether a tool earns its place. Whether a skill ever fires,
what it costs against what it returns, and the decision to remove it are
[`/qmw:audit-agent`](https://github.com/quietmachineworks/qmw), which reads
transcripts and weighs. agentlint states what is broken; audit-agent states what
is not worth carrying.

The line runs through shadowing rather than around it. That two descriptions
claim the same request is a measurement, and it is here. Which of the two should
be narrowed depends on which one has been firing, and that lives in the
transcripts.

It writes nothing, ever.

## What it reads

The user directory (`--config`), the repository's `.claude` (`--project`), the
platform's managed settings, and the plugins the agent actually loads.

That last one is not the plugin tree on disk. The cache keeps every version ever
fetched and the marketplace tree keeps every plugin ever offered, installed or
not, so discovery reads `installed_plugins.json` for the exact install path and
`enabledPlugins` for whether it is switched on. Walking the tree instead counts
an agent heavier than the one that starts and reports defects in files nothing
reads: on the machine this was built against, the difference was 48 phantom
skills and eight collisions that were stale versions of one plugin.

## Not yet

A hook priced against how often its matcher fires. Command-line settings passed
with `--settings`, which sit between managed and project in the stack.

## Layout

```
cmd/agentlint      the binary, flags and exit codes
internal/inventory discovery: settings, subagents, skills, following symlinks
internal/check     one type per check, plus the shared Finding
internal/schema    the published settings schema, embedded
internal/report    text for a terminal, JSON for a program
testdata/config    a configuration tree carrying one of every defect
```

## Contributing

[CONTRIBUTING.md](CONTRIBUTING.md) carries the checks to run and what a new
check owes. Security reports go through the Security tab, not a public issue:
[SECURITY.md](SECURITY.md).

## License

MIT
