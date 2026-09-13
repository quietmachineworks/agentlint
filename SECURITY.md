# Security

## What this program does

agentlint reads. It opens the settings files, subagent definitions and skill
definitions under one configuration directory, judges them, and prints what it
found. It writes no file, starts no process, and opens no network connection.
The schema it validates against is compiled into the binary, so a run needs
nothing fetched and works offline.

It is not a sandbox and does not pretend to be one. Pointing it at a
configuration directory means reading every file below it, including anything a
plugin left there.

## Credentials

An agent's configuration holds API keys: in `env`, in the environment a hook
carries, in an MCP server's definition. agentlint reads them, because it cannot
judge a file it has not read, and it never prints them. A finding on a field
whose name suggests a credential names the field and the shape of the problem,
never the value, and a test holds that.

Report the name, never the value, applies to anything built on top of this: the
JSON output follows the same rule, but a wrapper that dumps a config file
alongside it does not inherit the guarantee.

## Reporting a vulnerability

Open a private security advisory through the repository's Security tab. Please
do not open a public issue for a vulnerability.

Expect an acknowledgement within a week. This is a small project maintained by
one person, so a fix may take longer than an acknowledgement.
