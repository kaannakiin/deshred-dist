# deshred

**See Solana DEX swaps before they land.**

deshred listens to the raw shred traffic validators relay to each other on
Turbine — from any source that hands it those datagrams — rebuilds the
transactions while the block is still being assembled, and tells you which swaps
are about to hit which pools. Ten DEXes, four routers, one binary, and a
Jito-compatible gRPC stream you can plug an existing consumer into.

Where that traffic comes from and the four ways to feed it:
[docs/shred-feed.md](docs/shred-feed.md).

Closed source, free to run during the pilot. No payment system, no signup
form, no gatekeeping: download a build and take a key from
[PILOT-KEYS.md](PILOT-KEYS.md).

## What you actually get

- **Pre-confirmation transactions.** Same data a Jito shredstream gives you,
  reconstructed directly from UDP shreds on your own box. Leader signatures are
  checked so you are not fed forged slots.
- **A drop-in gRPC stream.** `SubscribeEntries`, wire-compatible with Jito's
  `shredstream.proto`. If your code already talks to a shredstream proxy, point
  it at deshred and it works. What crosses the wire is transactions —
  slot plus bincode `Vec<Entry>` — not decoded swaps. Exact shape, field by
  field: [docs/data-model.md](docs/data-model.md).
- **Venue decoders that are measured, not claimed.** Meteora DLMM / DAMM v1 /
  DAMM v2 / DBC, Orca Whirlpool, Raydium CLMM / AMM v4 / CPMM, pump.fun and
  PumpSwap, plus Jupiter v6, OKX, DFlow and Axiom router instructions. They run in `verify`,
  scored against confirmed blocks, and back the accuracy receipts below.
  Their live output is an opt-in service, `deshred.v1.SwapIntentStream`:
  one frame per venue swap leg — pool, side, the mints spent and received,
  mode, stated amount, stop-short flag — before the transaction executes.
  Router legs ride the same stream tagged with their router, for every hop whose
  pool the index already knows; `SubscribeSwapIntentChains` hands you the same
  legs one instruction per frame, and `deshred.v1.RouteStream` names every hop of
  a route plan, including the ones that produced no leg.
- **Opt-in pre-execution signals.** Another gRPC service, off by default, that
  carries what the pipeline works out on the way past: priority fee, Jito and
  Helius-Sender tip hits, a durable-nonce classifier, and — the part you cannot
  derive yourself — provenance, admission refusals, fork observations and arrival
  timing. The first three are a tested shortcut around rules that are easy to get
  wrong, not information the entry stream withholds; the rest is this pipeline's
  own findings. `SubscribeEntries` is unchanged either way.
  [docs/data-model.md](docs/data-model.md#pre-execution-signals-deshredv1-opt-in).
- **The schemas and working clients.** Every `.proto` this binary serves ships
  in [`proto/`](proto/) and as a release asset, with a codegen guide and
  runnable Rust, Go and TypeScript subscribers in [`examples/`](examples/).
  Tag numbers and enum values: [docs/proto-reference.md](docs/proto-reference.md).
- **Proof, not promises.** Every release ships an offline `verify` kit: one
  command replays a real captured slot and checks the decoder against what
  the confirmed block actually contained. It runs on your machine, needs no
  network, and takes under a minute.

## Who it is for

Searchers, market makers and analytics teams who want to see on-chain swap
flow as early as the network physically allows — without standing up a
validator, and without trusting a third-party feed they cannot audit.

## Get access

Download a build from [Releases](../../../releases) and take a license key from
[PILOT-KEYS.md](PILOT-KEYS.md) — 100 of them are published there. **No
application, no explanation of what you are building, no waiting on a reply.**

The key unlocks `run` (pass `--license <key>` or set `SHRED_LICENSE`). It is
checked locally against a public key inside the binary — nothing is sent
anywhere — and it never expires. `verify`, `replay` and `capture` need no key
at all, so you can audit the decoder before you run it.

What is asked in return is **feedback**: what worked, what broke, what was
missing, what the numbers looked like on your feed. Open an issue here or
message [@kaannakiin](https://t.me/kaannakiin). Quote the `sub` from your boot
log (`license accepted sub=pilot-042`) so a report can be told apart from
another pilot's.

## Run it (two minutes)

```sh
tar xzf deshred-<version>-x86_64-unknown-linux-gnu.tar.gz
cd deshred-<version>-x86_64-unknown-linux-gnu

# Unicast: point a jito-shredstream-proxy --dest-ip-ports at this host:7733,
# leave SHRED_GROUPS unset, and go.
export SHRED_LICENSE=<key from PILOT-KEYS.md>   # run is the only licensed command
./deshred run --rpc https://your-rpc-endpoint
```

That is the whole setup for the unicast path. On a multicast shred fabric, set
`SHRED_IFACE` and `SHRED_GROUPS` to what your provider assigned
([docs/multicast.md](docs/multicast.md)); every other source is in
[docs/shred-feed.md](docs/shred-feed.md).

`--rpc` is not optional in practice: without a leader schedule no slot can be
attributed to its leader, and the gRPC stream stays empty. Then consume the
stream from anywhere on the host:

```sh
tar xzf proto-kit-<version>.tar.gz          # the .proto files, also in proto/ here
grpcurl -plaintext \
  -import-path proto/jito-shredstream -proto shredstream.proto \
  127.0.0.1:9900 shredstream.ShredstreamProxy/SubscribeEntries
```

(This server serves no gRPC reflection, so grpcurl needs the schema — which is
why it ships. Details: [docs/grpc.md](docs/grpc.md).)

Full walkthrough: [docs/quickstart.md](docs/quickstart.md) ·
where the feed comes from: [docs/shred-feed.md](docs/shred-feed.md) ·
what you get back: [docs/data-model.md](docs/data-model.md) ·
gRPC details: [docs/grpc.md](docs/grpc.md) ·
every knob: [docs/env-reference.md](docs/env-reference.md).

## Is it accurate? Check for yourself

```sh
tar xzf verify-kit-<version>.tar.gz
./deshred verify --capture slice.dzcap --truth truth.json \
  --alt-journal alt-journal.jsonl --pools pools.json --expect expected.json
# -> VERIFY GREEN — matrices match expected.json
```

Green means the binary in your hands reproduces the accuracy this release was
measured at, on a real mainnet slot, offline. The same check gates every
release before it is published. What the numbers mean and how to run it
against your own captures: [docs/verify.md](docs/verify.md).

## What it will not do (read this)

- **It does not trade or quote.** It decodes. No pricing, no routing, no
  signing — there is no execution path in the binary.
- **It cannot see inside CPIs.** Shreds carry only top-level instructions. A
  swap wrapped inside another program's call is invisible to _any_ shred-level
  decoder; deshred counts those instead of hiding them. The router adapters
  recover what Jupiter/OKX state at the top level — that is as far as the data
  goes.
- **Pre-execution means pre-execution.** An intent can belong to a transaction
  that later reverts, or state a ceiling rather than a fill. `verify` separates
  those populations instead of blending them into one score.
- **Latency numbers are not published yet.** They were measured on reference
  hardware that is being recalibrated; this project does not ship numbers it
  cannot currently reproduce.

Everything above, in detail: [docs/limits.md](docs/limits.md).

## Platforms

| Target                     | Status                                                                                                       |
| -------------------------- | ------------------------------------------------------------------------------------------------------------ |
| `x86_64-unknown-linux-gnu` | **Primary.** The low-latency `run` path lives here (`recvmmsg`, busy-poll, core pinning).                    |
| macOS `x86_64` / `aarch64` | Works — `replay` and `verify` fully, `run` correct but not latency-tuned. Good for development and auditing. |
| musl                       | Not shipped until it is green in CI.                                                                         |

## Configuration at a glance

Everything is a flag or a same-named environment variable. The binary embeds no
endpoints, keys or secrets.

| Variable                   | Default          | What                                                                         |
| -------------------------- | ---------------- | ---------------------------------------------------------------------------- |
| `SHRED_PORT`               | `7733`           | UDP port the shreds arrive on                                                |
| `SHRED_GROUPS`             | _(empty)_        | Multicast groups; **empty = unicast**                                        |
| `SHRED_IFACE`              | _(none)_         | Interface carrying your multicast feed; required in multicast mode only      |
| `SHRED_RPC`                | _(none)_         | RPC for leader schedule / lookup tables — without it slots stay `unverified` |
| `SHRED_RPC_MAX_CONCURRENT` | `16`             | Cap on concurrent RPC calls; halves itself when your endpoint throttles      |
| `SHRED_GRPC_LISTEN_ADDR`   | `127.0.0.1:9900` | Where the stream is served                                                   |
| `SHRED_LICENSE`            | _(none)_         | Your personal key; required by `run` only, verified offline                  |

Full table, logging toggles, degraded-mode behaviour:
[docs/env-reference.md](docs/env-reference.md) ·
[docs/troubleshooting.md](docs/troubleshooting.md).

## License & contact

Free, closed-source binary under a proprietary [EULA](LICENSE) — use it,
don't redistribute it, no warranty. The pilot keys in
[PILOT-KEYS.md](PILOT-KEYS.md) are yours to take; reselling one is not allowed,
using one is. Third-party components and their licenses:
[THIRD-PARTY-LICENSES.txt](THIRD-PARTY-LICENSES.txt) (also inside every
tarball).

Questions, access, bugs: **Telegram [@kaannakiin](https://t.me/kaannakiin)** or
open an issue here. The issue template asks for your platform,
`deshred --version`, ingest mode, and whether `--rpc` was set — that is
usually enough to reproduce.
