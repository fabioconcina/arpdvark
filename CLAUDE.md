# arpdvark — Claude instructions

## Versioning and releases

- Always bump the version after implementing a change (before committing)
- "bump the version and commit" means: update the git tag (e.g. v0.3.0), commit all changes, tag
- "create a release" means: build both linux/amd64 and linux/arm64 via `make build-all`, then `gh release create`
- Always run tests before committing (see Build section for test commands)

## Build

- Always Linux only (`GOOS=linux`)
- `make build` → `./arpdvark` (local dev build)
- `make build-all` → `dist/arpdvark-linux-amd64` + `dist/arpdvark-linux-arm64`
- Tests: `go test ./tags/... ./vendor_db/... ./output/... ./exitcode/...` (no build tag needed); `go test -tags linux ./tui/... ./scanner/...` (Linux-only, run in CI)

## Deploying to server

- Always set the version via ldflags: `go build -ldflags "-X main.version=vX.Y.Z"`
- Never use bare `go build` without ldflags — the binary will show "dev" as the version

## Documentation

- When making user-facing changes (new features, changed flags, new CLI modes, changed behavior), always update both README.md and llms.txt to reflect the changes

## Communication style

- Keep responses short and direct
- No emojis
