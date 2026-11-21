## Fixes

- **The TUI no longer crashes when the table shrinks under the cursor.**
  With a row selected near the bottom, applying a filter, hiding offline
  devices or forgetting a device could leave the cursor past the last row,
  and the next `Enter`, `e` or `d` panicked. The cursor now moves to the
  last remaining row. This comes from the bubbles 1.0 table component.

## Verifiable releases

- **Binaries are built by GitHub Actions from the tagged commit.** Previous
  releases were built and uploaded by hand. The download names are
  unchanged (`arpdvark-linux-amd64`, `arpdvark-linux-arm64`), so existing
  install scripts keep working.

- **Checksums, SBOMs and provenance.** Each release now includes
  `checksums.txt`, an SPDX SBOM per binary, and a build provenance
  attestation you can check with
  `gh attestation verify arpdvark --repo fabioconcina/arpdvark`. The
  README has the steps.

- **Smaller binaries.** Release builds strip the symbol table and debug
  info.

## Dependencies

- **Updated the TUI and MCP libraries.** bubbletea 1.3.10, bubbles 1.0.0,
  lipgloss 1.1.0 and mcp-go 1.6.0, plus newer golang.org/x modules. The
  `--mcp` server exposes the same `scan_network` tool as before.
