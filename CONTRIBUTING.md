# Contributing

## What belongs here

agentlint resolves what an agent's configuration declares against what is
actually on the machine. Every check is deterministic: it reads files, compares
them to a schema or to the filesystem, and returns the same verdict every time.

Welcome:

- a check that catches a defect the runtime validator passes over, with the
  fixture that proves the gap
- a false positive, with the configuration that is actually valid
- a defect on a real machine that agentlint stayed quiet on
- a layout this tool fails to discover: a skill manager, an install method or an
  OS where the inventory comes back wrong

Not welcome: anything that needs a model to decide. Cost against usage, dormant
skills, description shadowing and the decision to remove something are
[`/qmw:audit-agent`](https://github.com/quietmachineworks/qmw), which reads
transcripts and weighs. The line between the two repositories is that one: this
binary states what is broken, that skill states what is not worth carrying.

Please open an issue before a large change, so the design discussion happens
before the work.

## Running the checks

```bash
go build ./...
go vet ./...
go test ./...
gofmt -l .          # prints nothing
go run honnef.co/go/tools/cmd/staticcheck@2025.1.1 ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
```

Every workflow uses GitHub's own actions and nothing else: this organization
allows only those and verified creators, and a workflow naming any other fails
before it starts. A tool that ships as a Go module needs no action at all.

The last two commands above are the only ones that touch the network. CI runs all of it on Linux, macOS
and Windows on every push and pull request, because path handling is most of the
risk here: a configuration tree is read by absolute path, and a skill manager
links one source tree into the agent's folder rather than copying it.

To run against a tree that is not yours:

```bash
go run ./cmd/agentlint --config testdata/config
```

## Adding a check

A check is a type with `Name() string` and `Run(*inventory.Inventory) []Finding`
in `internal/check`, added to `All()` in report order.

- a fixture under `testdata/config` carrying the defect, and a case in
  `internal/check/check_test.go` that fails before the check and passes after
- the fixture keeps the shape of the real thing: a folded YAML description, a
  symlinked skill folder, a variable only the runtime can expand
- an error is something that will not work. A warning is something suspect. When
  the evidence is a published schema rather than observed behaviour, prefer the
  warning and say what would settle it
- one defect is one finding. If another check already names it, drop it rather
  than print it twice
- no finding prints a value from a field that may hold a credential
- a line in `README.md` under Checks, and one in `CHANGELOG.md` under Unreleased

## Conventions

English everywhere in the tree. Comments explain a constraint a reader would
otherwise violate, and nothing else; if a comment states what the code is,
rename instead. No em dash anywhere, in code, prose, or commits: a comma, a
colon, a parenthesis, or two sentences. No attribution to an assistant, in files
or in commit trailers.

Commit subjects are one imperative sentence, capitalized, no type prefix and no
trailing period: `Run the published schema, which nothing else ever does`. A
release commit reads `Cut 0.2.0, <what changed since 0.1.0>`.

## Cutting a release

Move the Unreleased section of `CHANGELOG.md` under a dated heading, commit, tag
`vX.Y.Z`, push the tag. The release workflow builds the binaries and publishes
the changelog section as the release notes.
