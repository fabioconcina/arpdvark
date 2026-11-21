# arpdvark: Claude instructions

## Versioning and releases

- Always bump the version after implementing a change (before committing)
- Releases are cut by CI. To release vX.Y.Z:
  1. Write `RELEASE_NOTES.md` for the new version (it becomes the GitHub release body)
  2. Commit and push to `main`
  3. Push the tag: `git tag vX.Y.Z && git push origin vX.Y.Z`
- The `v*` tag triggers `.github/workflows/release.yml`, which runs goreleaser (linux amd64 + arm64 binaries, `checksums.txt`, SBOMs) and attests the checksums
- Never run goreleaser locally (beyond `goreleaser check`) and never use `gh release create`
- Always run tests before committing (see Build section for test commands)

## Build

- Always Linux only (`GOOS=linux`)
- `make build` → `./arpdvark` (local dev build, linux/amd64)
- `make build-all` → `dist/arpdvark-linux-amd64` + `dist/arpdvark-linux-arm64`
- `make test` (`go test -race -count=1 ./...`) and `make vet`
- `scanner` and `tui` tests carry `//go:build linux`, so on macOS they are skipped; CI runs them on Linux. Cross-check with `GOOS=linux go build ./... && GOOS=linux go vet ./...`

## Deploying to server

- Always set the version via ldflags: `go build -ldflags "-X main.version=vX.Y.Z"`
- Never use bare `go build` without ldflags: the binary will show "dev" as the version

## Documentation

- When making user-facing changes (new features, changed flags, new CLI modes, changed behavior), always update both README.md and llms.txt to reflect the changes

## Communication style

- Keep responses short and direct
- No emojis
