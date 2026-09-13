# agentlint

Resolve what an agent's configuration declares against what is actually on the
machine. Settings, hooks, subagents, skills. No model, no network, no tokens.

```bash
go install github.com/quietmachineworks/agentlint/cmd/agentlint@latest
agentlint
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

`--json` for another program, `--strict` to fail on warnings too, `--config` to
read a directory other than `CLAUDE_CONFIG_DIR` or `~/.claude`. Exit 0 clean,
1 with errors, 2 when the configuration directory cannot be read.

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

## What it does not do

It does not judge whether a tool earns its place. Cost against usage, dormant
skills, description shadowing and the decision to remove something are
[`/qmw:audit-agent`](https://github.com/quietmachineworks/qmw), which reads
transcripts and weighs. agentlint states what is broken; audit-agent states what
is not worth carrying.

It writes nothing, ever.

## Not yet

Full validation against the embedded schema, permission-rule breadth and
contradictions, cross-file precedence between user, project and policy settings,
MCP server resolvability, secrets found in plain text, and lexical shadowing
across descriptions.

## License

MIT
