# gh-axi — this fork's AXI layer

This repository is `onyx-space/cli`, a fork of [cli/cli](https://github.com/cli/cli)
that adds a **TOON (Token-Oriented Object Notation)** output layer for agent use,
installed as the `gh-axi` command.

## Why this fork exists

The previous `gh-axi` was a TypeScript wrapper around the `gh` binary. It had
three structural problems: it crashed inside pi sessions because `gh` emits
ANSI-colored JSON under color-forcing env vars (`CLICOLOR_FORCE`) that the
wrapper parsed with a bare `JSON.parse`; it re-implemented only ~15 commands so
agents saw mixed formats; and it shipped as two binaries with a runtime
`execFile` dependency. Forking `cli/cli` solves all three: TOON output is
written with plain `fmt` (ANSI-immune), the full gh command surface is
inherited, and one binary replaces two.

## What changed vs upstream

- Command identity renamed to `gh-axi` (executable path, root help, `--version`).
- Non-TTY output of converted commands emits TOON; TTY and `--json` output are
  byte-identical to upstream gh (pinned oracle: gh 2.96.0).
- `internal/toon` package escapes untrusted fields (quotes/commas/control
  chars) matching the `@toon-format/toon` encoder.
- Array commands emit a well-formed empty TOON state (`items[0]{...}` +
  `count: 0 of M`, exit 0) instead of gh's stderr `no items found`.
- `script/build-axi.sh` builds the `gh-axi` binary with the fork version
  injected and the auto-update check disabled (plain `go build` omits the
  `updateable` tag).

## Converted commands (TOON)

`repo view`, `repo list`, `issue list`, `issue view`, `pr list`, `pr view`,
`search repos`, `release list`, `run list`, `workflow list`.

## Build & install

```sh
make build-axi              # ./gh-axi, version 0.1.0-<commit>
make install-axi            # ~/.local/bin/gh-axi
VERSION=0.2.0 make build-axi
```

## Upstream sync

Fork strategy: `git fetch upstream` + merge on a cadence (per upstream minor
release; security-relevant releases within 7 days, owner: the m-machine
maintainer). Converted output branches are the merge-conflict surface — on
conflict, re-apply the TOON branch on top of upstream's new output code.
Custom work lives on the `trunk` branch.
