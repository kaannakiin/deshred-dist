# Environment reference

Every variable has a same-named CLI flag (`SHRED_PORT` ⇔ `--port`); the flag
wins when both are set. The one on/off variable, `SHRED_GRPC_SIGNALS`, needs an
explicit `true`/`false` in the environment, while its flag form is bare. The binary embeds no endpoints or secrets — the only
key inside it is the Ed25519 **public** key that checks your license locally —
and this table is the complete deployment surface.

## License

| Variable        | Default  | Meaning                                                                                                                                                                                                                          |
| --------------- | -------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `SHRED_LICENSE` | _(none)_ | A pilot license key (`dzl1.…`) — take one from [../PILOT-KEYS.md](../PILOT-KEYS.md). Required by **`run` only**; `replay`, `verify` and `capture` ignore it. Verified offline against the key embedded in the binary — no network call. Pilot keys never expire. |

A missing, malformed, expired or revoked key makes `run` exit before it opens
any socket, with a message naming the reason; see
[troubleshooting.md](troubleshooting.md#run-refuses-the-license-key).

## Ingest

| Variable          | Default   | Meaning                                                                                                                                                                                                      |
| ----------------- | --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `SHRED_IFACE`     | _(none)_  | Interface carrying your multicast feed. Read in multicast mode only, where it is **required** — `SHRED_GROUPS` set with no interface fails at boot rather than joining on `0.0.0.0`. Unicast never reads it. |
| `SHRED_PORT`      | `7733`    | UDP port for the shred feed.                                                                                                                                                                                 |
| `SHRED_GROUPS`    | _(empty)_ | Comma-separated multicast group(s). **Empty = unicast mode; set = multicast mode.** There is no separate mode flag.                                                                                          |
| `SHRED_RECV_CORE` | _(none)_  | CPU core to pin the receive thread. Linux only.                                                                                                                                                              |

## RPC

| Variable    | Default  | Meaning                                                                                                                                                                                                                                                                                                                                  |
| ----------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `SHRED_RPC` | _(none)_ | Solana RPC endpoint, resolved once at boot for the leader schedule, shred version, and address-lookup-table / mint fetches. Formally optional, but without it every slot stays `unverified` and **nothing is served** — `run` starts and the gRPC stream stays empty (see [limits.md](limits.md#degraded-mode-without-an-rpc-endpoint)). |

### RPC budget

`SHRED_RPC` is **your** endpoint and **your** quota — deshred runs on your
machine and never proxies through ours, so these knobs exist to let you cap what
this process spends. Defaults are deliberately modest; nothing here needs tuning
to get a correct stream.

| Variable                    | Default  | Meaning                                                                                                                                                                                                                                                                                   |
| --------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `SHRED_RPC_MAX_CONCURRENT`  | `16`     | Ceiling on concurrent JSON-RPC requests. The in-flight window starts at this number and halves whenever your endpoint answers `429`/`502`/`503`/`504`, times out, or drops the connection, then climbs back one at a time. `0` turns the governor off entirely — no ceiling, no back-off. |
| `SHRED_RPC_BATCH_MAX`       | `100`    | Pubkeys per `getMultipleAccounts` request. 100 is the protocol maximum; a larger value is refused at startup rather than silently truncated.                                                                                                                                              |
| `SHRED_RPC_BATCH_WINDOW_MS` | `20`     | How long a cache miss waits for company before its batch is sent. Lower means fresher, higher means fewer calls against your quota.                                                                                                                                                       |
| `SHRED_RPC_TIMEOUT_MS`      | `20000`  | Budget for one JSON-RPC call, covering the request and the response body together.                                                                                                                                                                                                        |
| `SHRED_RPC_NEGATIVE_TTL_MS` | `60000`  | How long an absent or refused account is left alone before it is asked for again.                                                                                                                                                                                                         |
| `SHRED_RPC_REFETCH_MIN_MS`  | `30000`  | Earliest an already-fetched key may be fetched again.                                                                                                                                                                                                                                     |
| `SHRED_RPC_AUTH_TOKEN`      | _(none)_ | API key for `SHRED_RPC`, sent as both `Authorization: Bearer …` and `x-token`. Environment only: the matching flag is hidden from `--help` so the value never lands in shell history, and no log line or config dump prints it.                                                           |
| `SHRED_RPC_FILL_QUEUE`      | `1024`   | Depth of each cache's fill queue. Requests past it are dropped rather than queued; the count appears as `alt_fill_dropped` on the `rpc health` log line.                                                                                                                                  |

Three flags have no environment form, matching each other:

| Flag             | Meaning                                                                                                                                                                                                                                                                     |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--alt-journal`  | JSONL journal of resolved address-lookup-table contents. Loaded at boot to warm-start the cache and appended on every fill, so a restart re-fetches nothing it already knows. Absent means every table is fetched again.                                                    |
| `--mint-journal` | The same, for raw Token-2022 mint accounts backing the transfer-fee/pause trait cache.                                                                                                                                                                                      |
| `--pool-journal` | The same, for the pool-facts index behind `SHRED_DECODE_INTENTS`. Three venues (Raydium AMM v4, Raydium CLMM's v1 `swap`, Meteora DAMM v1) name no mint in the instruction and resolve a side only once the pool is indexed; the journal keeps those fills across restarts. |

Every ten seconds `run` logs an `rpc health` line carrying the live window
(`rpc_limit_now`), the back-off counters, and both caches' fetch/refusal totals.

### Router pool probe

`run` only, and only with `SHRED_DECODE_INTENTS=true`. A route instruction names
the pools it will touch; when one of them is not in the index yet, the leg stands
down rather than guessing. The probe is the one surface in `run` that spends your
endpoint on accounts no venue swap asked about, so it is off by default. The
supported warm start is `router-coverage --rpc --harvested-pools` fed back as
`--pool-journal`.

| Variable                                  | Default    | Meaning                                                                                                                                                                                                                                                  |
| ----------------------------------------- | ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `SHRED_ROUTER_POOL_PROBE`                 | `false`    | Ask `SHRED_RPC` about the accounts a route named whose pool is not indexed. Same literal `true`/`false` rule as `SHRED_GRPC_SIGNALS`.                                                                                                                    |
| `SHRED_ROUTER_POOL_PROBE_MAX_PER_WINDOW`  | `50`       | Distinct probe keys admitted per fill window. Fills a venue swap actually named are never crowded out by these.                                                                                                                                          |
| `SHRED_ROUTER_POOL_PROBE_NEGATIVE_TTL_MS` | `21600000` | How long an account that resolved to "not a pool of any venue this build decodes" is remembered — six hours. Far longer than `SHRED_RPC_NEGATIVE_TTL_MS`: that one says a pool is not there _yet_; this one says the account is structurally not a pool. |

## gRPC output

| Variable                 | Default          | Meaning                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| ------------------------ | ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `SHRED_GRPC_LISTEN_ADDR` | `127.0.0.1:9900` | TCP listen address for `SubscribeEntries`.                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `SHRED_GRPC_UDS`         | _(none)_         | Also serve on this Unix domain socket. TCP keeps listening — this adds a listener rather than replacing one. Startup fails if the path cannot be bound.                                                                                                                                                                                                                                                                                                                               |
| `SHRED_DECODE_INTENTS`   | `false`          | Run the venue decoders on every dispatched transaction and, with `SHRED_GRPC_SIGNALS`, mount `deshred.v1.SwapIntentStream` (`SubscribeSwapIntents`, `SubscribeSwapIntentChains`) and `deshred.v1.RouteStream`. Off by default: on spends decode CPU here and, once the pool index fills itself, RPC budget on your endpoint. Same literal `true`/`false` rule as the flag below.                                                                                                      |
| `SHRED_GRPC_SIGNALS`     | `false`          | Mount the opt-in `deshred.v1.SignalStream` service on the same port and address. Off by default: the service still answers, with `Unimplemented` and a message naming this variable, and the pipeline does no per-transaction signal work at all. ⚠ Takes the literal `true` or `false` — `1`/`0`/`yes`/`no` are refused at startup. The CLI form is the bare flag `--grpc-signals`, which takes no value. See [data-model.md](data-model.md#pre-execution-signals-deshredv1-opt-in). |

## Logging

| Variable             | Default   | Meaning                                                     |
| -------------------- | --------- | ----------------------------------------------------------- |
| `SHRED_LOG_LEVEL`    | `info`    | Base level for all `dz::*` targets.                         |
| `SHRED_LOG_BOOT`     | _(unset)_ | Override for boot/startup logging.                          |
| `SHRED_LOG_TOPOLOGY` | _(unset)_ | Override for CPU pinning/topology logging.                  |
| `SHRED_LOG_FEED`     | _(unset)_ | Override for the ingest feed (includes the 10s stats line). |
| `SHRED_LOG_DECODE`   | _(unset)_ | Override for venue decoding.                                |
| `SHRED_LOG_ALT`      | _(unset)_ | Override for the address-lookup-table cache.                |
| `SHRED_LOG_STATS`    | _(unset)_ | Override for periodic statistics.                           |
| `SHRED_LOG_ANSI`     | `false`   | ANSI color on/off (`true`/`false`). Off by default.         |

## Measurement tools only

| Variable             | Used by    | Meaning                                                                            |
| -------------------- | ---------- | ---------------------------------------------------------------------------------- |
| `SHRED_TIP_ACCOUNTS` | `coverage` | Extra tip-account list for the coverage measurement subcommand. Not read by `run`. |
