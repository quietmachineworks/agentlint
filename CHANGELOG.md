# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `settings-scope`, and discovery of project and managed settings, so the
  precedence stack is the documented one rather than two files in a directory.
- Plugin-shipped hooks and MCP servers, read from the plugins the agent loads
  rather than from every plugin on disk.

- `mcp-unresolvable`, `secret-in-config`, `permission-rule`,
  `settings-precedence`, `name-collision` and `description-shadowing`. MCP
  servers are discovered where they actually live, which is beside the
  configuration directory rather than inside it.

- `settings-schema`, the whole settings file validated against the embedded
  published schema. Pattern mismatches are warnings, not errors, and a defect
  the dedicated checks already name is not reported twice.
- Five checks over settings, hooks, subagents and skills, run from one binary
  with no model and no network: `hook-unresolvable`, `settings-unknown-key`,
  `settings-unknown-hook-event`, `agent-frontmatter`, `skill-frontmatter`.
- The published Claude Code settings schema, embedded, read for the key list and
  the hook event list.
- `--json`, `--strict` and `--config`, with exit codes a CI job can branch on.
