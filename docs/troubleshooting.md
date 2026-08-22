# Troubleshooting

## Everything is `unverified`

Expected when `--rpc`/`DESHRED_RPC` is not set: without the leader schedule,
leader signatures cannot be checked, so no slot can be promoted to verified.
This is a deliberate degrade, not an error — set an RPC endpoint for verified
output. See [limits.md](limits.md#degraded-mode-without-an-rpc-endpoint).

## `run` refuses the license key

`run` is the only licensed command; it checks the key before touching any
socket, so a refusal has no side effects. The first line of the error says why:

- `license token is missing` — set `DESHRED_LICENSE` or pass `--license`.
- `license token is malformed` / `is not valid base64url` /
  `signature is N bytes, expected 64` / `is too large` — the key got mangled
  in transit (line break, missing segment, extra text pasted along). Paste it
  again as one line; leading/trailing whitespace is fine.
- `license token prefix is not supported by this build` — what you pasted is
  not a deshred key at all, or it was issued for a newer key generation than
  this release knows. Ask for a fresh one.
- `signature does not verify against this build's key` — the key was not
  issued for this binary (different issuer key, or edited). Ask for a fresh one.
- `license token expired at unix …` — your key carried an expiry; ask for a
  renewal.
- `license token … has been revoked` — the key was retired in this release.
  Contact the maintainer.

`replay`, `verify` and `capture` never ask for a key — if they do, you are
running something that is not deshred.

## The feed is silent (no datagrams in the stats line)

- **Unicast:** confirm your `jito-shredstream-proxy` `--dest-ip-ports` actually
  targets this host and this `DESHRED_PORT`, and that `DESHRED_GROUPS` is
  empty (a set group switches deshred into multicast mode and it will not read
  your unicast packets).
- **Multicast:** work through [multicast.md](multicast.md#troubleshooting-a-silent-feed) —
  interface name, group assignment, join state, firewall.
- A UDP port conflict fails at startup with a bind error; a _silent_ feed means
  the socket is fine and nothing is arriving on it.

## `verify` exits with "measured matrices differ from the pinned expectation"

No `VERIFY GREEN` line, non-zero exit. On the shipped verify-kit this means a
corrupt or mixed download: re-check
`checksums.txt` and the kit's `manifest.json` hashes, and make sure the binary
and the kit come from the same release tag. A mismatch on an intact,
same-release pair is a bug — report it with the full output.

## `verify` refuses my second `--archive-url`

By design. Different archives return materially different transaction sets for
the same slot, so ground truth must come from exactly one archive per run —
otherwise every ratio is corrupted. Pick one endpoint.

## Old flags are rejected

Retired measurement-era flags (`--unicast`, `--recv-buffer-bytes`,
`--busy-poll-us`, …) are rejected loudly instead of being silently ignored, so
stale deployment scripts fail fast instead of misconfiguring quietly. The
current surface is exactly what `deshred run --help` and
[env-reference.md](env-reference.md) show. In particular, there is no
`DESHRED_UNICAST`: unicast is simply `DESHRED_GROUPS` left empty.

## macOS: `run` works but seems slow

Expected. macOS builds decode correctly but the low-latency ingest path
(`recvmmsg`, busy-polling, core pinning) is Linux-only; macOS is a
development/verification platform. `replay` and `verify` are fully supported
there. See [limits.md](limits.md).

## Reporting a bug

Open an issue with: platform and OS version, `deshred --version`, ingest mode
(unicast/multicast), whether `--rpc`/`DESHRED_RPC` was set, and the relevant
log lines. For decode-accuracy claims, a `verify` run against a single archive
endpoint is the strongest possible report.
