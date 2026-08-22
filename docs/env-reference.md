# Environment reference

Every variable has a same-named CLI flag (`DESHRED_PORT` ⇔ `--port`); the flag
wins when both are set. The binary embeds no endpoints or secrets — the only
key inside it is the Ed25519 **public** key that checks your license locally —
and this table is the complete deployment surface.

## License

| Variable          | Default  | Meaning                                                                                                                                                                                                              |
| ----------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `DESHRED_LICENSE` | _(none)_ | Your personal license key (`dzl1.…`). Required by **`run` only**; `replay`, `verify` and `capture` ignore it. Verified offline against the key embedded in the binary — no network call. Does not expire unless the key says so. |

A missing, malformed, expired or revoked key makes `run` exit before it opens
any socket, with a message naming the reason; see
[troubleshooting.md](troubleshooting.md#run-refuses-the-license-key).

## Ingest

| Variable            | Default       | Meaning                                                                                                             |
| ------------------- | ------------- | ------------------------------------------------------------------------------------------------------------------- |
| `DESHRED_IFACE`     | `doublezero1` | Network interface to bind.                                                                                          |
| `DESHRED_PORT`      | `7733`        | UDP port for the shred feed.                                                                                        |
| `DESHRED_GROUPS`    | _(empty)_     | Comma-separated multicast group(s). **Empty = unicast mode; set = multicast mode.** There is no separate mode flag. |
| `DESHRED_RECV_CORE` | _(none)_      | CPU core to pin the receive thread. Linux only.                                                                     |

## RPC

| Variable      | Default  | Meaning                                                                                                                                                                                                                    |
| ------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `DESHRED_RPC` | _(none)_ | Solana RPC endpoint, resolved once at boot for the leader schedule, shred version, and address-lookup-table / mint fetches. Optional — without it `run` starts in degraded `unverified` mode (see [limits.md](limits.md)). |

## gRPC output

| Variable                   | Default          | Meaning                                    |
| -------------------------- | ---------------- | ------------------------------------------ |
| `DESHRED_GRPC_LISTEN_ADDR` | `127.0.0.1:9900` | TCP listen address for `SubscribeEntries`. |
| `DESHRED_GRPC_UDS`         | _(none)_         | Unix domain socket path instead of TCP.    |

## Logging

| Variable               | Default   | Meaning                                                     |
| ---------------------- | --------- | ----------------------------------------------------------- |
| `DESHRED_LOG_LEVEL`    | `info`    | Base level for all `dz::*` targets.                         |
| `DESHRED_LOG_BOOT`     | _(unset)_ | Override for boot/startup logging.                          |
| `DESHRED_LOG_TOPOLOGY` | _(unset)_ | Override for CPU pinning/topology logging.                  |
| `DESHRED_LOG_FEED`     | _(unset)_ | Override for the ingest feed (includes the 10s stats line). |
| `DESHRED_LOG_DECODE`   | _(unset)_ | Override for venue decoding.                                |
| `DESHRED_LOG_ALT`      | _(unset)_ | Override for the address-lookup-table cache.                |
| `DESHRED_LOG_STATS`    | _(unset)_ | Override for periodic statistics.                           |
| `DESHRED_LOG_ANSI`     | `false`   | ANSI color on/off (`true`/`false`). Off by default.         |

## Measurement tools only

| Variable               | Used by    | Meaning                                                                            |
| ---------------------- | ---------- | ---------------------------------------------------------------------------------- |
| `DESHRED_TIP_ACCOUNTS` | `coverage` | Extra tip-account list for the coverage measurement subcommand. Not read by `run`. |
