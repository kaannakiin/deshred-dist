# deshred

**See Solana DEX swaps before they land.**

deshred listens to the raw shred traffic validators exchange — over a
DoubleZero multicast feed or a plain `jito-shredstream-proxy` relay — rebuilds
the transactions while the block is still being assembled, and tells you which
swaps are about to hit which pools. Ten DEXes, two routers, one binary, and a
Jito-compatible gRPC stream you can plug an existing consumer into.

Closed source, free to run during the pilot. No payment system, no signup
form — just [message me on Telegram](https://t.me/kaannakiin).

## What you actually get

- **Pre-confirmation transactions.** Same data a Jito shredstream gives you,
  reconstructed directly from UDP shreds on your own box. Leader signatures are
  checked so you are not fed forged slots.
- **Decoded swap intents**, not raw bytes. Pool, direction, amounts for
  Meteora DLMM / DAMM v1 / DAMM v2 / DBC, Orca Whirlpool, Raydium CLMM / AMM v4
  / CPMM, pump.fun and PumpSwap — plus swap intent lifted out of Jupiter v6 and
  OKX router instructions.
- **A drop-in gRPC stream.** `SubscribeEntries`, wire-compatible with Jito's
  `shredstream.proto`. If your code already talks to a shredstream proxy, point
  it at deshred and it works.
- **Proof, not promises.** Every release ships an offline `verify` kit: one
  command replays a real captured slot and checks the decoder against what
  the confirmed block actually contained. It runs on your machine, needs no
  network, and takes under a minute.

## Who it is for

Searchers, market makers and analytics teams who want to see on-chain swap
flow as early as the network physically allows — without standing up a
validator, and without trusting a third-party feed they cannot audit.

## Get access

Builds are handed out personally during the pilot:

**Telegram → [@kaannakiin](https://t.me/kaannakiin)** — tell me roughly what
you are building and which platform you run on; you get the download link, a
**personal license key** and setup help directly.

The key unlocks `run` (pass `--license <key>` or set `DESHRED_LICENSE`). It is
checked locally against a public key inside the binary — nothing is sent
anywhere — and it does not expire unless yours says so. `verify`, `replay` and
`capture` need no key at all, so you can audit the decoder before you ever ask
for one.

## Run it (two minutes)

```sh
tar xzf deshred-<version>-x86_64-unknown-linux-gnu.tar.gz
cd deshred-<version>-x86_64-unknown-linux-gnu

# Unicast: point a jito-shredstream-proxy --dest-ip-ports at this host:7733,
# leave DESHRED_GROUPS unset, and go.
export DESHRED_LICENSE=<your-key>   # from Telegram; run is the only licensed command
./deshred run --rpc https://your-rpc-endpoint
```

That is the whole setup for the unicast path — no DoubleZero seat needed. Have
a DoubleZero multicast feed? Set `DESHRED_IFACE` and `DESHRED_GROUPS` to what
your provider gave you ([docs/multicast.md](docs/multicast.md)).

Then consume the stream from anywhere on the host:

```sh
grpcurl -plaintext 127.0.0.1:9900 shredstream.ShredstreamProxy/SubscribeEntries
```

Full walkthrough: [docs/quickstart.md](docs/quickstart.md) ·
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
| `DESHRED_PORT`             | `7733`           | UDP port the shreds arrive on                                                |
| `DESHRED_GROUPS`           | _(empty)_        | Multicast groups; **empty = unicast**                                        |
| `DESHRED_IFACE`            | `doublezero1`    | Interface to bind (multicast)                                                |
| `DESHRED_RPC`              | _(none)_         | RPC for leader schedule / lookup tables — without it slots stay `unverified` |
| `DESHRED_GRPC_LISTEN_ADDR` | `127.0.0.1:9900` | Where the stream is served                                                   |
| `DESHRED_LICENSE`          | _(none)_         | Your personal key; required by `run` only, verified offline                  |

Full table, logging toggles, degraded-mode behaviour:
[docs/env-reference.md](docs/env-reference.md) ·
[docs/troubleshooting.md](docs/troubleshooting.md).

## License & contact

Free, closed-source binary under a proprietary [EULA](LICENSE) — use it,
don't redistribute it, no warranty; your license key is personal and
non-transferable. Third-party components and their licenses:
[THIRD-PARTY-LICENSES.txt](THIRD-PARTY-LICENSES.txt) (also inside every
tarball).

Questions, access, bugs: **Telegram [@kaannakiin](https://t.me/kaannakiin)** or
open an issue here. The issue template asks for your platform,
`deshred --version`, ingest mode, and whether `--rpc` was set — that is
usually enough to reproduce.
