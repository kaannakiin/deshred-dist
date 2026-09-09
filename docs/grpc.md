# Consuming the gRPC feed

`deshred run` serves reconstructed entries over gRPC using the same wire
contract as Jito's shredstream proxy: service `shredstream.ShredstreamProxy`,
method `SubscribeEntries`, from
[jito-labs/mev-protos](https://github.com/jito-labs/mev-protos) `shredstream.proto`
(vendored at a pinned commit, Apache-2.0). An existing Jito shredstream
consumer connects with no client-side changes — same proto, same stream shape.

Every schema this server speaks is shipped: [`proto/`](../proto/) in this
repository and `proto-kit-<version>.tar.gz` on each release, with a compile and
codegen guide in [proto/README.md](../proto/README.md) and runnable clients in
[examples/](../examples/). The commands below assume you are standing in a
directory that has `proto/` beside you.

## Endpoint

| Setting                  | Default          | Meaning                                             |
| ------------------------ | ---------------- | --------------------------------------------------- |
| `SHRED_GRPC_LISTEN_ADDR` | `127.0.0.1:9900` | TCP listen address.                                 |
| `SHRED_GRPC_UDS`         | _(unset)_        | Serve on a Unix domain socket at this path instead. |

The default binds loopback only — put your own proxy/TLS in front if you need
to expose it beyond the host.

## Smoke test

```sh
grpcurl -plaintext \
  -import-path proto/jito-shredstream -proto shredstream.proto \
  127.0.0.1:9900 shredstream.ShredstreamProxy/SubscribeEntries
```

**This server does not serve the gRPC reflection API.** A bare
`grpcurl -plaintext 127.0.0.1:9900 list` fails with `server does not support the
reflection API` — that is expected, not a broken deployment, and it is why every
command on this page carries the `-import-path`/`-proto` pair. The upside is
that `list` and `describe` need no server at all:

```sh
grpcurl -import-path proto/deshred-v1 \
        -proto signals.proto -proto intents.proto -proto routes.proto list
# deshred.v1.RouteStream
# deshred.v1.SignalStream
# deshred.v1.SwapIntentStream

grpcurl -import-path proto/deshred-v1 -proto intents.proto \
        describe deshred.v1.SwapIntentSignal
```

Each stream message carries a slot number and serialized ledger entries; the
entries deserialize with the standard Solana entry format, exactly as they do
from a Jito shredstream endpoint. The bytes are forwarded without re-encoding,
so they are byte-identical to what that proxy would emit for the same slot.

`SubscribeEntriesRequest` is an empty message: there are no server-side filters,
and every subscriber receives the same complete stream. Filtering is
client-side.

The full contract — every field, how to unpack `entries`, and what is
deliberately absent — is [data-model.md](data-model.md).

## Trust semantics

Entries are published only from slots that passed admission. When no RPC
endpoint is configured, leader signatures cannot be checked and slots are
`unverified` — see [limits.md](limits.md#degraded-mode-without-an-rpc-endpoint)
before consuming a degraded feed for anything that matters.

## Pre-execution signals (opt-in)

`SHRED_GRPC_SIGNALS=true` mounts a second service, `deshred.v1.SignalStream`, on
the same address. It carries per-transaction facts (priority fee, tip hits,
durable-nonce witness, and the ALT-resolved account key list) and per-slot
observations (provenance and admission, slot poison, forks, arrival latency)
that `SubscribeEntries` does not.

| Service                        | Mounted by                       |
| ------------------------------ | -------------------------------- |
| `shredstream.ShredstreamProxy` | always                           |
| `deshred.v1.SignalStream`      | `SHRED_GRPC_SIGNALS=true`        |
| `deshred.v1.SwapIntentStream`  | …and `SHRED_DECODE_INTENTS=true` |
| `deshred.v1.RouteStream`       | same pair of knobs               |

```sh
grpcurl -plaintext -import-path proto/deshred-v1 -proto signals.proto \
  127.0.0.1:9900 deshred.v1.SignalStream/SubscribeTxSignals

grpcurl -plaintext -import-path proto/deshred-v1 -proto signals.proto \
  127.0.0.1:9900 deshred.v1.SignalStream/SubscribeSlotSignals
```

With the flag off the service still answers — with `Unimplemented` and a message
naming the variable, rather than an absent method you would have to guess about.

## Swap intents (opt-in, two knobs)

`SHRED_GRPC_SIGNALS=true` together with `SHRED_DECODE_INTENTS=true` mounts a third
service, `deshred.v1.SwapIntentStream`, carrying one frame per venue swap leg the
decoders read out of a transaction before it executes: venue, pool, side, the
mints it spends and receives, mode, stated amount and whether the instruction
states a stop-short bound.

```sh
grpcurl -plaintext -import-path proto/deshred-v1 -proto intents.proto \
  127.0.0.1:9900 deshred.v1.SwapIntentStream/SubscribeSwapIntents
```

The refusal names both variables, because signals that are on and intents that
are off look identical from outside. Three venues (Raydium AMM v4, Raydium CLMM's
v1 `swap`, Meteora DAMM v1) need the pool account to name a side; `run
--pool-journal <path>` keeps those fills across restarts the way `--alt-journal`
does for lookup tables. Field by field, and what an intent does and
does not promise: [data-model.md](data-model.md#subscribeswapintents--stream-of-swapintentsignal).

## Swap intent chains (same two knobs)

`SubscribeSwapIntentChains`, on the same `deshred.v1.SwapIntentStream` service,
carries the same legs grouped by the instruction that produced them: one frame
per venue or router instruction, every leg of it inside, `leg_index` ascending.
A direct venue swap is a chain of one. A `chained` leg is sized by the leg
before it, so a consumer that sizes legs reads this rpc instead of reassembling
a route from per-leg frames that may interleave with other transactions.

```sh
grpcurl -plaintext -import-path proto/deshred-v1 -proto intents.proto \
  127.0.0.1:9900 deshred.v1.SwapIntentStream/SubscribeSwapIntentChains
```

The two rpcs are two queues over one set of legs; each numbers its own frames,
and a leg inside a chain carries the chain's `publish_seq`. Chain frames are not
ordered by transaction execution — sort on `(slot, tx_index, tx_ix_index)` if you
need that. Field by field:
[data-model.md](data-model.md#subscribeswapintentchains--stream-of-swapintentchain).

## Route frames (same two knobs)

`deshred.v1.RouteStream/SubscribeRoutes` is the router's side of the story: one
frame per route instruction naming every hop the plan carried, in plan order —
the hops that became legs (`claimed`, with the `leg_index` to join on), the
hops on a venue this build does not decode (`unmapped`, with the router's own
venue byte), and the hops that stood down (`unresolved`, with the reason). A
leg-only stream cannot say that a route had five hops and this engine claimed
two; this one does.

```sh
grpcurl -plaintext -import-path proto/deshred-v1 -proto routes.proto \
  127.0.0.1:9900 deshred.v1.RouteStream/SubscribeRoutes
```

Join a leg to its route on `(slot, tx_index, tx_ix_index, leg_index)`. Field by
field: [data-model.md](data-model.md#subscriberoutes--stream-of-routesignal).

## Ordering across streams

Every rpc is its own queue. None is ordered against another or against
`SubscribeEntries`; join on `(slot, signature)` and use each stream's
`publish_seq` to spot dropped frames. The field-by-field contract, including
what an absent priority fee means, is in
[data-model.md](data-model.md#pre-execution-signals-deshredv1-opt-in).
