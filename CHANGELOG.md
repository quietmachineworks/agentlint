# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-09-14

### Added

- `hook-cost`, the price of a matcher per matching call: how many hooks sit on
  it and the worst case they add up to, with the runtime's 60s default for an
  entry that declares none. A warning, never an error, and only past a floor
  of three hooks or two minutes, because every configuration has one hook on
  `Bash`.
- `skill-lock`, the `skills` CLI lockfile read against its tree: a skill edited
  in place since install, which the next sync overwrites; an entry with nothing
  installed behind it; a link into the manager's tree that the lockfile does
  not know. The hash recomputed is the git tree object the CLI records for a
  GitHub source, CRLF undone. A machine without the lockfile reads nothing here.
- `--settings`, the file or inline JSON the agent is given as `claude
  --settings`, read at its published place in the stack, between managed and
  project local, so `settings-precedence` reports what it overrides.
- A Homebrew tap, written by goreleaser on every tagged release:
  `brew install quietmachineworks/tap/agentlint`.

### Changed

- `settings-schema` no longer reports a pattern mismatch under `permissions`.
  The published pattern rejects any rule whose argument holds a closing
  parenthesis; the runtime accepts one, settled against it on 2026-09-14:
  `Bash(touch 'out (1).txt')` and the escaped form the runtime writes itself,
  `Bash(touch 'out \(1\).txt')`, each given as `--settings`, allowed exactly
  that command where it was denied without the rule. The schema is not
  authoritative there.

## [0.1.1] - 2026-09-13

### Fixed

- A binary from `go install` reported its version as "dev". Only a release build
  carries the version as a linker flag, so the module version it was built from
  now stands in when that flag is absent.
- The 0.1.0 release was published with empty notes. Disabling the changelog told
  goreleaser to empty the release body rather than leave the file the workflow
  passes to fill it.

## [0.1.0] - 2026-09-13

First release.

### Added

- Thirteen checks over an agent's configuration, run from one binary with no
  model, no network and no tokens. A published JSON schema already describes
  `settings.json` and `claude plugin validate` already judges plugin manifests;
  neither answers whether any of it is still true, which is what this does.
- Hooks and MCP servers resolved against the machine: a command that is missing,
  is a directory, or is not executable. A command whose target only the runtime
  can expand is reported as unverifiable, never as broken.
- The whole settings file validated against the published schema, which is
  embedded. Pattern mismatches are warnings rather than errors, because a regex
  in a published schema is the likeliest place for it to simplify the parser it
  describes.
- Top-level keys one edit away from a real setting, and hooks keyed on an event
  that will never fire. The schema tolerates unknown keys, so nothing else
  catches a typo there.
- `settings-scope`: a key sitting at a level that does not honour it, which the
  schema states per key and nothing else reports.
- `settings-precedence` over the published stack: managed, project local, shared
  project, user. A file with no published place in that stack is read and never
  used to claim another decides nothing.
- Permission rules judged against each other: an allowance covering a whole
  tool, a rule another already covers, a rule a denial has taken back.
- Credentials found in settings and MCP files, named by field and shape and
  never by value.
- Frontmatter of skills and subagents, including the fields the runtime
  validator passes over. Subagents it does not discover at all.
- One name installed twice, and two descriptions that are the same text.
- Discovery that follows what the agent actually loads: symlinked skill trees, a
  folded YAML description, and the plugins named in `installed_plugins.json`
  rather than every version and listing on disk.
- `--json`, `--strict`, `--config` and `--project`, with exit codes a CI job can
  branch on. It writes nothing, ever.
