# deshred

**Offline decoding and verification for Solana shred traffic.**

deshred consumes raw Agave shred UDP traffic — from a DoubleZero multicast feed
or a [`jito-shredstream-proxy`](https://github.com/jito-labs/shredstream-proxy)
unicast relay — reassembles it into ledger entries and transaction facts before
the transactions land on chain, decodes swap activity across 10 DEX venues plus
2 swap routers, and re-publishes the result as a Jito-compatible
`SubscribeEntries` gRPC stream. It ships as a single closed-source binary with
an offline `verify` command, so you can check its output against a real
confirmed block without trusting a word of this README.

## What deshred is not

- **Not a quoting or trading engine.** It decodes swap intents; it never prices
  a pool, builds a route, or signs a transaction. There is no execution path in
  this binary.
- **Not omniscient about CPIs.** Shreds carry only pre-execution top-level
  instructions. A swap that happens inside a cross-program invocation (e.g. a
  Raydium swap invoked from inside an aggregator route) is structurally
  invisible to shred-level decoding — a property of the data source, not a bug
  budget. deshred counts what it cannot see instead of pretending otherwise.
  Two built-in router adapters (Jupiter v6, OKX DEX Router) partially work
  around this by reading the router's own top-level instruction data, not by
  seeing inside CPIs.
- **Not a price or execution guarantee.** Decoded intents are read before the
  transaction executes. Whether it landed, and whether it moved the pool the
  way stated, is exactly what `verify`'s matrices measure — see
  [Prove the binary](#prove-the-binary-verify).

## Venue coverage

10 DEX venues are decoded end-to-end (pool identity, swap direction, amounts):
Meteora DLMM, Meteora DAMM v1, Meteora DAMM v2, Meteora DBC, Orca Whirlpool,
Raydium CLMM, Raydium AMM v4, Raydium CPMM, pump.fun, PumpSwap.

9 of those 10 are **quotable** (they carry a resolvable pool with reserves):
everything above except pump.fun, whose bonding-curve accounts have no pool to
quote against — its swaps are still decoded, just never priced. Note that
quoting itself is not shipped in this binary (see
[What deshred is not](#what-deshred-is-not)); "quotable" describes the decoded
data, not a feature.

Two additional router adapters (Jupiter v6, OKX DEX Router) extract swap intent
directly from the router's own top-level instruction bytes, independent of the
venue list above.

## Supported platforms

| Target                      | Status                       | Notes                                                                                                                                                  |
| --------------------------- | ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `x86_64-unknown-linux-gnu`  | **Primary, fully supported** | The only target the low-latency `run` path is supported on — `recvmmsg`, busy-polling, and core pinning are Linux-only.                                |
| macOS (`x86_64`, `aarch64`) | Degraded                     | `replay` and `verify` work fully. `run` decodes correctly but is **not** latency-competitive — treat macOS as a development and verification platform. |
| musl (any arch)             | Not shipped                  | Not promised until proven green in CI.                                                                                                                 |

## Quickstart (unicast via jito-shredstream-proxy)

The fastest path to a working feed does not require DoubleZero access. Point an
existing [`jito-shredstream-proxy`](https://github.com/jito-labs/shredstream-proxy)
`--dest-ip-ports` at this host's `DESHRED_PORT`, leave `DESHRED_GROUPS`
**unset**, and deshred binds a plain unicast socket — no multicast join, no
group to guess.

```sh
tar xzf deshred-<version>-x86_64-unknown-linux-gnu.tar.gz
cd deshred-<version>-x86_64-unknown-linux-gnu

# DESHRED_GROUPS left empty -> unicast mode (there is no separate "unicast" flag)
./deshred run --rpc https://your-rpc-endpoint
```

Point your proxy at `<this-host>:7733`, or set `DESHRED_PORT` to match your
proxy's destination port. Details: [docs/quickstart.md](docs/quickstart.md).

## Multicast (DoubleZero)

Set `DESHRED_IFACE` to your DoubleZero interface name and `DESHRED_GROUPS` to
the multicast group(s) your provider assigned you (comma-separated). deshred
does not ship a group default tuned to any specific provider — enter what your
provider gives you:

```sh
export DESHRED_IFACE=<your-doublezero-interface>
export DESHRED_GROUPS=<your-assigned-multicast-groups>
./deshred run --rpc https://your-rpc-endpoint
```

deshred warns (but does not refuse) if a group falls outside the documented
`233.84.178.0/24` DoubleZero range — that usually means it is worth
double-checking with your provider. Details: [docs/multicast.md](docs/multicast.md).

## Prove the binary (`verify`)

Download `verify-kit-<version>.tar.gz` from the release you installed — it
bundles a real captured shred slice plus the ground truth decoded from that
slot's confirmed block — and run:

```sh
tar xzf verify-kit-<version>.tar.gz
./deshred verify \
  --capture slice.dzcap --truth truth.json --alt-journal alt-journal.jsonl \
  --pools pools.json --expect expected.json
```

Expected output:

```text
VERIFY GREEN — matrices match expected.json
```

That means the binary you downloaded reproduces the same accuracy matrices this
project measured and pinned at release time. No network access is required —
the fixture is fully offline. Every published artefact passed this exact check
in the release pipeline before it was allowed to ship. Matrix definitions and
interpretation: [docs/verify.md](docs/verify.md).

### Measured, receipted

On the slot the fixture is drawn from (mainnet slot 440061516), the pipeline
reconstructed 2,213 transactions from raw shreds before execution; 676 were
votes. The remaining 1,537 matched a public archive's confirmed non-vote
transaction set **exactly** — same set, same order, zero difference in either
direction. Across a larger corpus (15 capture windows, 928 slots, ~1.03M
transactions), FEC recovery succeeded on 293,521 of 293,521 attempted
recoveries (100%).

⚠ Throughput and latency numbers are **not published** in current releases —
the reference hardware they were measured on is being recalibrated, and this
project does not ship numbers it cannot currently reproduce.

## Consuming the gRPC feed

`deshred run` re-publishes reconstructed entries as a `SubscribeEntries` stream
wire-compatible with [Jito Labs' `shredstream.proto`](https://github.com/jito-labs/mev-protos)
(vendored at a pinned commit, Apache-2.0). Existing Jito shredstream consumers
connect with no client-side changes.

```sh
grpcurl -plaintext 127.0.0.1:9900 list
# shredstream.ShredstreamProxy
```

Listen address defaults to `127.0.0.1:9900` (`DESHRED_GRPC_LISTEN_ADDR`); a
Unix domain socket is available via `DESHRED_GRPC_UDS`. Details:
[docs/grpc.md](docs/grpc.md).

## Configuration

Everything is CLI flags or same-named environment variables — the binary embeds
no endpoints, no keys, no secrets.

| Variable                   | Default          | Meaning                                                              |
| -------------------------- | ---------------- | -------------------------------------------------------------------- |
| `DESHRED_IFACE`            | `doublezero1`    | Network interface to bind.                                           |
| `DESHRED_PORT`             | `7733`           | UDP port for the shred feed.                                         |
| `DESHRED_GROUPS`           | _(empty)_        | Comma-separated multicast group(s). **Empty = unicast.**             |
| `DESHRED_RECV_CORE`        | _(none)_         | CPU core to pin the receive thread (Linux only).                     |
| `DESHRED_RPC`              | _(none)_         | Solana RPC endpoint. Optional — see [Degraded mode](#degraded-mode). |
| `DESHRED_GRPC_LISTEN_ADDR` | `127.0.0.1:9900` | gRPC listen address.                                                 |
| `DESHRED_GRPC_UDS`         | _(none)_         | Unix domain socket path instead of TCP.                              |
| `DESHRED_LOG_LEVEL`        | `info`           | Base log level.                                                      |

There is no `DESHRED_UNICAST` variable — unicast is derived automatically
whenever `DESHRED_GROUPS` is empty. Full reference including per-subsystem log
toggles: [docs/env-reference.md](docs/env-reference.md).

## Degraded mode

`deshred run` starts even without `--rpc`/`DESHRED_RPC` — it does not refuse to
run. Without an RPC endpoint the leader schedule is unknown, so leader
signatures cannot be checked and every slot stays `unverified`; that state is
counted and visible, never silently dropped. Set `--rpc` for verified output.
See [docs/limits.md](docs/limits.md) for the full list of honest limits.

## Version pins

Each release states the exact `solana-ledger` version its shred wire format and
FEC recovery were built and verified against (currently `2.3.13`), plus the
pinned commit of the vendored Jito proto. Pins are bumped deliberately,
per-release, with the verify fixture re-proven — see the release notes of the
tag you are running.

## CLI reference

`deshred --help` lists all 11 subcommands. The four you will use as a consumer:

- `deshred run` — start the live feed.
- `deshred replay` — decode a captured `.dzcap`/`.pcap` file offline.
- `deshred verify` — prove a binary's decode accuracy against a confirmed block.
- `deshred capture` — record raw shred traffic to a `.dzcap` file.

Of the remaining seven, `probe` is network diagnostics for the raw-shred feed
(transport smoke test, group/port discovery); the other six (`coverage`,
`enrich`, `promote`, `router-coverage`, `snapshot`, `venue-census`) are
measurement tools used to build the fixtures `verify` checks against.
`deshred <subcommand> --help` if curious.

## License

Free, closed-source binary under a proprietary [EULA](LICENSE) — no
redistribution, no warranty, use at your own risk. Third-party components
(MIT/Apache-2.0 decoders, the vendored Jito proto) are listed with full license
texts in [THIRD-PARTY-LICENSES.txt](THIRD-PARTY-LICENSES.txt), shipped inside
every release tarball.

## Support

This repository ships binaries and documentation only — no source, no CI. File
bugs and questions via Issues; the template asks for your platform,
`deshred --version`, ingest mode (unicast/multicast), and whether
`--rpc`/`DESHRED_RPC` was set.
