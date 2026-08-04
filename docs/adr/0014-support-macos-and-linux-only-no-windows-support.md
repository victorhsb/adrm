---
kind: adr
id: ADR-0014
title: Support macOS and Linux only; no Windows support
status: accepted
date: 2026-08-04
tags: cli, go, platforms
supersedes: 
superseded_by: 
deprecated_by: 
---
# ADR-0014: Support macOS and Linux only; no Windows support

## Status

accepted

## Context

canon is a Go CLI distributed as a single binary. Supporting an OS means testing path handling, file permissions, shell integration, and install scripts on it, plus answering user reports for it. The maintainers run and verify on macOS and Linux; Windows brings real portability costs (path separators, permission models, shell behavior, install script design) with no maintainer able to verify it. Reasonable projects could support Windows via Go's cross-compilation, but untested cross-compiled binaries ship hidden failures to users.

## Decision

The supported platforms are macOS and Linux. Windows is not a supported platform: the project does not promise the CLI, install scripts, or tests work there, does not cross-compile Windows release artifacts, and does not accept bug reports specific to Windows. CI and install scripts target macOS and Linux only. This decision may be revisited if a maintainer willing to verify Windows appears; Go's portability means code may still build and run there, but without support guarantees.

## Consequences

Install scripts and shell-facing code may rely on POSIX semantics (sh, forward-slash paths, Unix file permissions) without Windows fallbacks; contributors are not expected to test on Windows. CI runs on Linux and macOS only, so Windows-specific regressions are invisible by design. Release artifacts are built for darwin and linux targets only. Windows users are not blocked from building from source — Go's portability means the code may compile and run there — but issues specific to Windows are closed as unsupported. If Windows support is ever adopted, it requires a superseding ADR plus CI coverage and an owner for Windows verification.

## History: Accepted

Date: 2026-08-04

Stated platform support policy; macOS and Linux are the verified targets.
