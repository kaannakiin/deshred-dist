---
name: Bug report
about: Something misbehaved — a crash, a wrong decode, a silent feed
labels: bug
---

## Environment (required)

- Platform / OS version (e.g. `x86_64-unknown-linux-gnu`, Ubuntu 22.04):
- `deshred --version`:
- Ingest mode: unicast (`DESHRED_GROUPS` empty) / multicast
- `--rpc` / `DESHRED_RPC` set: yes / no (degraded mode changes expected behavior — see docs/limits.md)

## What happened

<!-- Exact command, exact output. Quote errors verbatim. -->

## What you expected

## Logs / evidence

<!--
Relevant log lines (DESHRED_LOG_LEVEL=debug helps).
For decode-accuracy claims: the strongest report is a `deshred verify` run
against a single archive endpoint — include its output.
Reminder: a CPI-wrapped swap being invisible is a documented structural limit
(docs/limits.md), not a bug.
-->
