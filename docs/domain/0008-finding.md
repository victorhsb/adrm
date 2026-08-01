---
kind: domain
id: DM-0008
title: Finding
status: accepted
date: 2026-08-01
tags: glossary
supersedes: 
superseded_by: 
deprecated_by: 
---
# DM-0008: Finding

## Status

accepted

## Definition

A single problem or advisory reported by validating the corpus: a check name, a severity (error or warning), an optional location (path and/or id), and a suggested fix. Healthy (ok) results are diagnostics, not findings.

## Avoid

- **diagnostic** — a diagnostic is any validation check result including ok entries; a finding is only the error/warning subset
- **error** — a finding may be a warning, not necessarily an error
- **issue** — an issue implies a tracker item, a finding is a validation output
- **violation** — a violation implies a rule breach, some findings are advisory

## Relationships

Findings are produced by validating the [Corpus](0007-corpus.md), at [Shallow](0009-shallow-validation.md) or [Deep](0010-deep-validation.md) depth.
