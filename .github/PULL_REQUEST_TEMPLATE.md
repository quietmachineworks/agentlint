## What changes

## Why

If this adds or changes a check: what a run said before, what it says after, and
why the new answer is the true one.

## Checks

- [ ] `go build ./...`, `go vet ./...` and `go test ./...` pass
- [ ] `gofmt -l .` prints nothing
- [ ] a check change has a fixture under `testdata/` and a case that fails before it and passes after
- [ ] a new check is named in `README.md` and in `CHANGELOG.md` under Unreleased
- [ ] no finding prints a value from a field that may hold a credential
- [ ] no em dash, no attribution trailer
