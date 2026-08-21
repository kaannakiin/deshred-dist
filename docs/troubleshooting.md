# Troubleshooting

## Everything is `unverified`

Expected when `--rpc`/`DESHRED_RPC` is not set: without the leader schedule,
leader signatures cannot be checked, so no slot can be promoted to verified.
This is a deliberate degrade, not an error — set an RPC endpoint for verified
output. See [limits.md](limits.md#degraded-mode-without-an-rpc-endpoint).

## The feed is silent (no datagrams in the stats line)

- **Unicast:** confirm your `jito-shredstream-proxy` `--dest-ip-ports` actually
  targets this host and this `DESHRED_PORT`, and that `DESHRED_GROUPS` is
  empty (a set group switches deshred into multicast mode and it will not read
  your unicast packets).
- **Multicast:** work through [multicast.md](multicast.md#troubleshooting-a-silent-feed) —
  interface name, group assignment, join state, firewall.
- A UDP port conflict fails at startup with a bind error; a _silent_ feed means
  the socket is fine and nothing is arriving on it.

## `verify` prints RED / a matrix mismatch

On the shipped verify-kit this means a corrupt or mixed download: re-check
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
