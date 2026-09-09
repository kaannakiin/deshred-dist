# What you receive

This page is the complete data contract: what deshred takes in, what it hands
back, and exactly which knobs change either. If you are migrating from a
Yellowstone gRPC consumer, read [What you can turn on and off](#what-you-can-turn-on-and-off)
first — the filtering model is the biggest difference.

## Where deshred taps the pipeline

A Solana validator moves a transaction through four stages: **gossip** (the
leader receives it), **shredding** (it is split into UDP packets and broadcast),
**execution** (it runs, producing logs, fees and balances), and the **Geyser
hook** (the finished result is handed to plugins).

deshred reads stage 2. Yellowstone gRPC reads stage 4. Everything that follows
is a consequence of that one difference.

|                          | deshred                         | Jito shredstream      | Yellowstone gRPC                       |
| ------------------------ | ------------------------------- | --------------------- | -------------------------------------- |
| Extraction point         | Shreds, pre-execution           | Shreds, pre-execution | Geyser hook, post-execution            |
| Transaction `meta`       | Not available                   | Not available         | Complete                               |
| Inner (CPI) instructions | Not available                   | Not available         | Complete                               |
| Execution status         | Not known yet                   | Not known yet         | Known                                  |
| Server-side filters      | None                            | None                  | Rich (`accountInclude`, commitment, …) |
| Leader signature checked | **Yes**                         | No                    | N/A (post-consensus)                   |
| Equivocation handling    | **Slot poisoned, never served** | No                    | N/A                                    |
| Runs on your own box     | Yes                             | Yes                   | No (hosted RPC)                        |

The last three rows are the reason to run deshred rather than consume a shred
feed directly: every FEC set's Merkle root is verified against the leader
schedule before its transactions are served, and a slot that arrives in two
differently-signed versions is dropped rather than forwarded.

## What goes in

deshred never dials Jito, or anything else, for its transaction feed. It reads
**raw UDP datagrams** off a socket you point at it.

| Source                                                        | Transport                                                 | Configure with                 |
| ------------------------------------------------------------- | --------------------------------------------------------- | ------------------------------ |
| `jito-shredstream-proxy`                                      | UDP unicast to `<your-host>:7733` (its `--dest-ip-ports`) | leave `SHRED_GROUPS` empty     |
| Your own node on the Turbine tree, forwarding raw shred UDP   | UDP unicast to `<your-host>:7733`                         | leave `SHRED_GROUPS` empty     |
| A multicast shred fabric (DoubleZero is the deployed example) | UDP multicast                                             | `SHRED_IFACE` + `SHRED_GROUPS` |
| A capture file (pcap or DZCAP3)                               | none — `deshred replay`                                   | `--capture <file>`             |

Every live source carries the same bytes — unframed Agave shreds, plus a proxy's
own heartbeat datagrams, which deshred recognises and discards. The two socket
modes are the only place the difference exists; the full picture is in
[shred-feed.md](shred-feed.md).

One side channel exists: `SHRED_RPC` is a normal Solana JSON-RPC endpoint,
read for the leader schedule and shred version at boot, and for address-lookup-table
and Token-2022 mint accounts on demand. It carries no transaction data.

## What comes out

One surface, wire-compatible with Jito's shredstream proxy. The proto is
vendored verbatim at a pinned commit (`jito-labs/mev-protos`, Apache-2.0):

```proto
service ShredstreamProxy {
  rpc SubscribeEntries(SubscribeEntriesRequest) returns (stream Entry);
}

message SubscribeEntriesRequest {
  // tbd: we may want to add filters here
}

message Entry {
  // the slot that the entry is from
  uint64 slot = 1;

  // Serialized bytes of Vec<Entry>
  bytes entries = 2;
}
```

That is the whole message. Two fields:

| Field     | Type     | Meaning                                               |
| --------- | -------- | ----------------------------------------------------- |
| `slot`    | `uint64` | The slot these entries belong to.                     |
| `entries` | `bytes`  | bincode-serialized `Vec<solana_entry::entry::Entry>`. |

`entries` is **not re-encoded**. It is the exact payload the shred reassembler
produced, forwarded untouched, so it is byte-identical to what a Jito
shredstream proxy would emit for the same slot. A test pins that byte-identity
on every build.

### Unpacking `entries`

The payload is bincode with Solana's standard configuration: fixed-width
little-endian integers, and a `u64` element count in front of the outer vector.
Each element is:

```rust
struct Entry {
    num_hashes: u64,
    hash: Hash,                            // [u8; 32]
    transactions: Vec<VersionedTransaction>,
}
```

Each `VersionedTransaction` is encoded exactly as a transaction is on the wire —
`short_vec` (compact-u16) length prefixes, the `0x80`-flagged version byte on v0
messages — so **any existing Solana transaction parser reads it unchanged**. In
Rust that is one call:

```rust
let entries: Vec<solana_entry::entry::Entry> = bincode::deserialize(&msg.entries)?;
for entry in &entries {
    for tx in &entry.transactions {
        // tx.signatures, tx.message.header, tx.message.static_account_keys(),
        // tx.message.recent_blockhash(), tx.message.instructions(),
        // tx.message.address_table_lookups()
    }
}
```

### The shape you get, field by field

Parsed for readability — the stream itself carries the bytes described above.

```jsonc
{
  "slot": 440061516,
  "entries": [
    // bincode Vec<Entry>, decoded here
    {
      "numHashes": 12500,
      "hash": "6bqQ…", // PoH tick hash
      "transactions": [
        {
          "signatures": [
            "53boczQpTxg2fpKp4mDtadM4aHghw7Jhci6CHsyNpNBouuNoXESFP4UvBADxYHx1V1M9ghKWgeuyeL5UJED7H5s3",
          ],
          "message": {
            "header": {
              "numRequiredSignatures": 1,
              "numReadonlySignedAccounts": 0,
              "numReadonlyUnsignedAccounts": 8,
            },
            "accountKeys": ["…"], // static keys only
            "recentBlockhash": "…",
            "instructions": [
              {
                "programIdIndex": 8,
                "accounts": [7, 2, 9, 3, 1, 4, 0, 6, 5],
                "data": "…", // raw instruction bytes
              },
            ],
            "addressTableLookups": [
              // present, NOT resolved
              {
                "accountKey": "…",
                "writableIndexes": [3, 7],
                "readonlyIndexes": [1],
              },
            ],
          },
        },
      ],
    },
  ],
  // no meta            — no logs, no inner instructions, no compute units consumed
  // no pre/postBalances, no pre/postTokenBalances
  // no err / status    — execution has not happened yet
  // no blockTime, no blockhash for the slot
}
```

Vote transactions are **included** — deshred serves what the leader broadcast,
and votes are a large share of it: on the slot shipped with the verify kit
(440061516), 676 of 2213 reconstructed transactions were votes. Filter them
client-side on the `Vote111111111111111111111111111111111111111` program id if
you do not want them.

`addressTableLookups` are forwarded as the transaction states them. Resolving a
lookup index to a real account key needs the table's on-chain contents, and
`SubscribeEntries` carries the unresolved indexes, exactly as Jito's does — the
byte-identity of that stream is not negotiable.

deshred keeps that cache internally for its own decoding, and since it has the
resolved list anyway, the opt-in signal stream hands it to you: see
`account_keys_packed` under [pre-execution
signals](#pre-execution-signals-deshredv1-opt-in). That is the one place the
resolved form is available; it does not change what `SubscribeEntries` sends.

## What you can turn on and off

**On the subscription: nothing.** `SubscribeEntriesRequest` is an empty message
— upstream's own comment calls filters a "tbd" — and the server ignores the
request object entirely. There is no `accountInclude`, no `accountExclude`, no
`accountRequired`, no `vote`/`failed` toggle, no `commitment`, no `from_slot`.
Every subscriber receives the same complete stream and filters client-side.

This is the one place a Yellowstone consumer must change code: the filtering you
push to the server there happens in your process here.

**On deployment: two variables**, pinned by a test so the surface cannot grow
silently.

| Variable                 | Default          | Effect                                                                                                                                                                  |
| ------------------------ | ---------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `SHRED_GRPC_LISTEN_ADDR` | `127.0.0.1:9900` | TCP listen address.                                                                                                                                                     |
| `SHRED_GRPC_UDS`         | _(unset)_        | **Adds** a Unix-domain-socket listener. TCP keeps listening; this does not replace it. If the path cannot be bound, startup fails rather than quietly serving TCP only. |

There is no TLS and no authentication — the default bind is loopback. Put your
own proxy in front if the stream must leave the host. `TCP_NODELAY` is set on
every accepted connection.

**What governs stream content:** `SHRED_RPC`. Without it there is no leader
schedule, so no FEC set can be attributed to its scheduled leader, and nothing
is admitted — **the gRPC stream stays empty**. `run` still starts, and the logs
still count what arrived, but no subscriber receives a frame. See
[limits.md](limits.md#degraded-mode-without-an-rpc-endpoint).

## Delivery and backpressure

Each subscriber gets its own 256-frame queue. When a consumer falls behind, the
**oldest frame is dropped** and a counter increments; the subscriber keeps
streaming. This is a deliberate divergence from `jito-shredstream-proxy`, which
ends a lagging subscriber's stream instead. A slow consumer here degrades rather
than disconnects — check your own gaps in `slot` if that matters to you.

## Pre-execution signals (`deshred.v1`, opt-in)

Off by default. Set `SHRED_GRPC_SIGNALS=true` and two extra streams appear on the
same port, under service `deshred.v1.SignalStream`. Add `SHRED_DECODE_INTENTS=true`
and the decoders' output appears beside them: `deshred.v1.SwapIntentStream` (two
rpcs, legs and per-instruction chains) and `deshred.v1.RouteStream`.
`SubscribeEntries` is untouched in every case: same bytes, same shape, same queue.

**Be clear on what half of this is.** Priority fee, tip hits and the
durable-nonce witness are each a pure function of bytes `SubscribeEntries`
already hands you. They carry no information you could not derive yourself. What
they save you is the CPU to re-parse every transaction deshred has already
parsed, and the correctness of three rules that are easy to get wrong:

- the priority fee is charged on the **requested** compute-unit limit, and when a
  transaction states a price but no limit the runtime does not charge zero — it
  derives a default that depends on live cluster state (see the gap field below);
- Jito and Helius-Sender tips are ordinary System transfers, identifiable only
  against an allowlist of 18 accounts;
- a durable-nonce transaction is defined by the runtime as instruction 0 being a
  System `AdvanceNonceAccount` **whose first account is writable** — the
  writability check is the part hand-rolled detectors drop.

Provenance, fork and arrival-latency signals are a different thing entirely:
they are this pipeline's own findings — leader-signature verification,
equivocation detection, arrival timing — and cannot be derived from the published
bytes at any price.

### `SubscribeTxSignals` → stream of `TxSignal`

| Field                                 | Meaning                                                                                                                                                                                                               |
| ------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `slot`, `tx_index`                    | Position. `tx_index` is the transaction's index within the slot.                                                                                                                                                      |
| `signature`                           | Raw 64 bytes, not base58. Join key against the entry stream.                                                                                                                                                          |
| `priority_fee_lamports`               | `ceil(cu_price × cu_limit / 1e6)`. **Present only when `priority_fee_gap` is `NONE`.**                                                                                                                                |
| `priority_fee_gap`                    | Why there is no number. See the table below — an absent fee is never simply "zero".                                                                                                                                   |
| `cu_limit`, `cu_price_micro_lamports` | As stated by the transaction. Absent when it stated neither.                                                                                                                                                          |
| `tip_hits`                            | Top-level System transfers to a known Jito or Helius-Sender tip account: provider, lamports, ix index.                                                                                                                |
| `nonce_account`                       | Present ⇒ durable-nonce transaction, and this is the account it would advance.                                                                                                                                        |
| `alt_resolution`                      | Always set. `NOT_APPLICABLE` / `RESOLVED` / `MISS` / `STALE` — see below.                                                                                                                                             |
| `account_keys_packed`                 | The full account key list in runtime order, as concatenated 32-byte pubkeys. See below.                                                                                                                               |
| `token22_traits`                      | One entry per distinct Token-2022 mint the transaction moves: availability, both transfer-fee terms with their epochs, pause/hook/delegate flags and `blocks_transfer`. Account state — not derivable from the bytes. |
| `venue_touches`                       | One entry per venue instruction at top level (`tx_ix_index` set) plus one per venue reached only under a CPI (`tx_ix_index` absent): venue, event kind, pool when named, `decoded`. What was touched, never how much. |
| `scheduler_cost_units`                | Agave's **admission-time** cost estimate for this transaction. Absent when `cost_gap` says why. See below.                                                                                                            |
| `cost_gap`                            | Why there is no cost number. An absent cost is never zero.                                                                                                                                                            |
| `writable_account_mask`               | Bitmask over `account_keys_packed`: the accounts the runtime will actually write-lock, after demotions. See below.                                                                                                    |
| `writability_gap`                     | Why the mask is empty, when it is.                                                                                                                                                                                    |
| `publish_seq`                         | Monotonic per-stream counter. A missing number means a frame was dropped; see below.                                                                                                                                  |

#### Resolved account keys

`account_keys_packed` is the complete key list the runtime would load: the
static keys, then every lookup's writable indexes, then every lookup's readonly
indexes. An instruction's account indexes index it directly. Keys are packed
flat — key `i` is bytes `[i*32, i*32+32)` and the count is `len/32` — because on
a measured mainnet slot two transactions in three carry lookups at ~30 resolved
keys each, and a `repeated bytes` field would have meant tens of thousands of
small allocations per slot for every subscriber.

Without this you would run your own lookup-table resolver against historical
state, including the same-slot-extend rule that makes a naive resolver wrong.
With it, that step and its RPC calls leave your process entirely.

⚠ It is present **only** when `alt_resolution` is `RESOLVED` and the transaction
actually used a lookup table. It is empty otherwise, and the reason is in
`alt_resolution` rather than in the emptiness:

| `alt_resolution` | What it means                                                                                        |
| ---------------- | ---------------------------------------------------------------------------------------------------- |
| `NOT_APPLICABLE` | The transaction used no lookup table. Its keys are already complete in the `SubscribeEntries` bytes. |
| `RESOLVED`       | `account_keys_packed` is the runtime-order list.                                                     |
| `MISS`           | The table is not in this deployment's cache yet; a fetch was queued. Warm again shortly.             |
| `STALE`          | An index reached past the cached copy of the table; a refetch was queued.                            |

`MISS` and `STALE` are facts about **your** RPC endpoint and cache warmth, not
about the chain. A cold start reports them until the cache fills; on a measured
slot a warm cache answered 974 of 1002 lookup-carrying transactions with no RPC
call at all. `--alt-journal` carries that warmth across restarts.

| `priority_fee_gap` | What it means                                                                                                                                                                                                             |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `NONE`             | `priority_fee_lamports` is exact.                                                                                                                                                                                         |
| `NO_PRICE_STATED`  | No `SetComputeUnitPrice`. A real zero.                                                                                                                                                                                    |
| `LIMIT_UNSTATED`   | A price with no limit. The runtime charges against a default derived from the instruction mix **and a live feature set**, so it is not a function of the transaction bytes. Measured at 1.05% of fee-paying transactions. |
| `INVALID`          | Malformed or duplicated ComputeBudget instruction — the transaction fails on-chain, so no fee applies.                                                                                                                    |

#### Cost and write locks

Two fields that are easy to misread, so read this before using either.

`scheduler_cost_units` is what `CostTracker::try_add` charges against every
writable account before deciding one more transaction fits the block:
signature + write-lock + instruction-data + requested-CU + loaded-accounts-data
terms. It is an **admission estimate, not settled cost** — the runtime refunds
the difference as each transaction completes. Per transaction it is exact and
verified against the chain (mainnet slot 440061516: 1,451 transactions, zero
violations). Summed over a whole slot it is only an upper bound, and measurably
loose: 349,477,578 estimated against 69,833,922 settled on that slot, 5.00×. Do
not compare a slot total against a ceiling — and note deshred publishes no
ceiling, because the per-account write limit is not gated in the source and no
RPC returns it. The numerator is ours; the ratio is yours to choose.

`writable_account_mask` is the set the runtime actually locks. Two demotions
apply on top of what the transaction asked for: the runtime's reserved keys, and
an account the transaction itself invokes as a program (absent an upgradeable
loader). **This is not the population Agave prices.** Its write-lock cost term
is `300 × the positional writable count` — what the transaction _requested_ —
which is a different number. Cost goes with the request; attribution goes with
the mask. Measured: they differ on 9 of 2,213 transactions (0.46%).

#### Token-2022 mint traits

Each `token22_traits` entry is one distinct Token-2022 mint the transaction
moves, read from account state:

| Field                                                                                               | Meaning                                                                                                                                                                     |
| --------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `mint`                                                                                              | The mint's 32 bytes.                                                                                                                                                        |
| `availability`                                                                                      | Whether the traits were read, and if not, why.                                                                                                                              |
| `transfer_fee_older`, `transfer_fee_newer`                                                          | The two fee configurations a Token-2022 mint carries, each `basis_points` + `maximum_fee` + the `epoch` it takes effect in. Which one applies depends on the current epoch. |
| `paused`, `non_transferable`, `default_state_frozen`, `permanent_delegate`, `confidential_transfer` | Extension flags as set on the mint.                                                                                                                                         |
| `transfer_hook`, `transfer_hook_program`                                                            | Whether a transfer hook is configured, and the program it points at.                                                                                                        |
| `blocks_transfer`                                                                                   | The rolled-up verdict: this mint's state would stop a transfer outright.                                                                                                    |

⚠ These are **account state, and account state changes.** A trait read one slot
can be stale the next; that is a real staleness axis, unlike pool mint/vault
pairs which are fixed after init. Fee-mint legs matter more than their frequency
suggests: measured by mint, the amount an instruction declares is wrong on 34 of
51 legs entering a pool (67%), and 0 of those 51 instructions declare the fee at
all — only the balance table sees it. By population, though, such swaps are
rare: 5 of 18,495 top-level venue swaps (0.027%).

### `SubscribeSlotSignals` → stream of `SlotSignal`

One `slot`, one `publish_seq`, and one of five observations:

- **`provenance`** — `recovered` / `verified` / `poisoned`, the transaction count
  (`txs`), and `admitted`. ⚠ `admitted: false` is the interesting case: an unverified or
  poisoned batch reaches neither the entry stream nor the decoders, so this is the
  only place a refusal is ever visible.
- **`poisoned`** — the slot carries two independently signed versions, with the
  reason (`FEC_ROOT_CONFLICT`, `LEADER_SIG_INVALID`, …).
- **`fork`** — two slots claim the same parent, naming both (`parent_slot`, the
  shared parent; `sibling_slot`, the other claimant).
- **`slot_complete`** — the slot is done: `last_index`, the final shred index
  when it was seen, and `reason`, why deshred stopped waiting for more.
- **`latency`** — `batch_wait_us`, `decode_us`, `early_forward_us`. These are
  this process's own timings, in microseconds; they say whether _your feed_ is
  behaving, not whether the chain is.

### `SubscribeSwapIntents` → stream of `SwapIntentSignal`

Mounted only when **both** `SHRED_GRPC_SIGNALS=true` and `SHRED_DECODE_INTENTS=true`.
One frame per venue swap leg, read from the instruction bytes before execution.

⚠ **An intent is the instruction's claim, not the chain's outcome.** The
transaction may revert (measured on the shipped slot: 81 of 196 intents sat on
transactions that later failed), a stop-short bound may fill less than the stated
amount, and a Token-2022 fee moves the vault by `amount_pool_side`, not by the
stated number. Everything under a CPI stays invisible — for the three
concentrated-liquidity venues that is five swaps in six.

| Field                                                    | Meaning                                                                                                                                                                                                                                                                                                             |
| -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `slot`, `tx_index`, `signature`                          | Join keys against `TxSignal` and the entry stream.                                                                                                                                                                                                                                                                  |
| `tx_ix_index`, `leg_index`                               | Which top-level instruction, and which leg of it (two-hop and router shapes carry several).                                                                                                                                                                                                                         |
| `dex`, `pool`                                            | Venue and pool account. Always set.                                                                                                                                                                                                                                                                                 |
| `a_to_b` **or** `direction_gap`                          | Exactly one. The side relative to the pool's own mint order, or the named reason there is none — see the table below. A gap is never "no swap".                                                                                                                                                                     |
| `mode`                                                   | `EXACT_IN`, `EXACT_OUT`, or `PARTIAL_FILL` — the last states a **ceiling** the program may fill only part of.                                                                                                                                                                                                       |
| `exact_amount` / `chained` / `unsized`                   | Exactly one. `chained` is a leg sized by the leg before it; `unsized` is one the instruction never stated.                                                                                                                                                                                                          |
| `limit`, `sqrt_price_limit_le16`, `price_impact_cap_bps` | As stated. `sqrt_price_limit_le16` is a little-endian u128; zero means "no bound", which several venues hard-code.                                                                                                                                                                                                  |
| `states_a_stop_short`                                    | The instruction names a bound the program may stop at before spending `exact_amount`. ⚠ `false` is not "fills whole": Orca and Raydium CLMM truncate at the absolute price bound with nothing recording it, and a price-impact cap reverts rather than truncates.                                                   |
| `input_mint`, `output_mint`                              | What the swap spends and receives. Both absent whenever the side is a gap.                                                                                                                                                                                                                                          |
| `input_mint_traits`, `output_mint_traits`                | The same read-only Token-2022 lookups as `TxSignal.token22_traits`, for these two mints.                                                                                                                                                                                                                            |
| `amount_pool_side`                                       | What the pool side moves for `exact_amount` once a Token-2022 fee on the mode-pinned mint applies (exact-in: credited net; exact-out: debited gross). Present only from a fresh trait entry with a live fee. ⚠ Never applied to the other side.                                                                     |
| `router`, `cyclic`                                       | `DIRECT` for a venue instruction at top level; otherwise which router assembled the leg, and whether the route ends in the token it started in.                                                                                                                                                                     |
| `route_hop`                                              | Present on a router leg: which hop of the plan produced it — `hop_index`, the plan's token slots (`input_slot` → `output_slot`, slot 0 being the route input) and the share of that input slot this hop takes (`share_num`/`share_den`). Joins the leg to the matching hop on `RouteStream` without re-deriving it. |
| `publish_seq`                                            | Monotonic per-stream counter.                                                                                                                                                                                                                                                                                       |

| `direction_gap`    | What it means                                                                                                                                                  |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `POOL_UNKNOWN`     | The venue's instruction names no mint, and this deployment's pool index does not hold the pool yet. Raydium AMM v4, Raydium CLMM's v1 `swap`, Meteora DAMM v1. |
| `FACTS_INCOMPLETE` | The pool is indexed but lacks the field this venue's witness needs.                                                                                            |
| `MINT_MISMATCH`    | The instruction names a mint or vault the indexed pool does not carry — the index is wrong for this pool, or the swap cannot land.                             |
| `ATA_UNRESOLVED`   | The user's token accounts are neither canonical ATAs nor accounts the transaction itself created, so no side can be told.                                      |

Every other venue — PumpSwap, Meteora DLMM / DAMM v2 / DBC, Orca, Raydium CLMM
`swap_v2` and CPMM — names its mints in the instruction, so those intents resolve
with **no pool index at all**. Measured over 560 mainnet slots, that is 66,720 of
67,410 intents (99.0%); the remaining 1.0% ship as `POOL_UNKNOWN` until the pool
index learns the pool.

### `SubscribeSwapIntentChains` → stream of `SwapIntentChain`

Same service, same two knobs, same legs — grouped by the instruction that
produced them. One frame per venue or router instruction; a direct venue swap is
a chain of one.

| Field                           | Meaning                                                                                                                                                                                    |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `slot`, `tx_index`, `signature` | Join keys, as on the leg.                                                                                                                                                                  |
| `tx_ix_index`                   | The instruction every leg in this frame came from. `(slot, tx_index, tx_ix_index)` is the chain's key on both intent rpcs and on `RouteStream`.                                            |
| `router`                        | `DIRECT` for a venue instruction, otherwise the router that assembled the legs.                                                                                                            |
| `legs`                          | Every `SwapIntentSignal` the decoder produced for this instruction, `leg_index` ascending. A hop that stood down is not a leg and is not here — `SubscribeRoutes` names it.                |
| `publish_seq`                   | Monotonic on this rpc's own counter. Every leg inside the frame carries this same number, so a leg copied out of its frame still names the frame; the per-leg rpc numbers legs on its own. |

Why it exists: a `chained` leg is sized by the leg before it, and on the per-leg
rpc the legs of one instruction can interleave with other transactions' frames.
Read this rpc when you size legs; read the per-leg rpc when you only need to
know a pool is about to be touched.

Chain frames are not ordered by transaction execution: the legs of one
instruction stay together, but the instructions of one transaction may interleave
with other transactions' frames. Sort on `(slot, tx_index, tx_ix_index)` if you
need execution order.

### `SubscribeRoutes` → stream of `RouteSignal`

`deshred.v1.RouteStream`, same two knobs. One frame per route **instruction** —
not per transaction, not per leg — naming every hop the plan carried, in plan
order, including the hops that produced no leg. A leg-only stream structurally
cannot say that a route had five hops and this engine claimed two.

| Field                                          | Meaning                                                                                                                                                                                      |
| ---------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `slot`, `tx_index`, `signature`, `tx_ix_index` | Join keys. A leg joins its route on `(slot, tx_index, tx_ix_index, leg_index)` against a `claimed` hop.                                                                                      |
| `router`                                       | Which router: Jupiter v6, OKX, DFlow or Axiom.                                                                                                                                               |
| `mode`, `source_mint`, `dest_mint`             | The route's own statement. ⚠ `mode` is the one field on any stream that may arrive `UNSPECIFIED`: a plan refused before it was parsed far enough has no mode, and `plan_stand_down` says so. |
| `stated_amount`, `min_out`                     | Read as `mode` says: an exact-in route states its input, an exact-out route its output, a token-ledger route neither.                                                                        |
| `platform_fee_bps`                             | What the router skims. ⚠ Which side it comes off is not in the instruction, so a route that charges one seeds no leg amount.                                                                 |
| `runtime_chosen_venue`, `cyclic`               | The plan names a venue chosen at run time; the route ends in the token it started in.                                                                                                        |
| `hops[]`                                       | `hop_index`, the route's own token slots (`input_slot` → `output_slot`; slot 0 is the route input), the share of the input slot this hop takes, and exactly one outcome:                     |
| &nbsp;&nbsp;`claimed`                          | The hop became a leg: `leg_index`, `dex`, `pool`.                                                                                                                                            |
| &nbsp;&nbsp;`unmapped`                         | The router wanted a venue this build does not decode; `raw_venue_tag` is the router's own byte.                                                                                              |
| &nbsp;&nbsp;`unresolved`                       | The hop stood down: `dex` and the reason.                                                                                                                                                    |
| `plan_stand_down`                              | Set only when the whole plan could not be enumerated — without it an empty `hops` would read like a plan whose every hop stood down.                                                         |
| `publish_seq`                                  | Monotonic on this rpc's own counter.                                                                                                                                                         |

### What you are and are not guaranteed

- **`TxSignal` is not globally ordered.** Frames are published from several decode
  workers at once. If you need order, buffer and key on `(slot, tx_index)`.
- **`SlotSignal` is produced in order** by a single thread — but a lagging
  subscriber still loses frames, so received order is not delivery order.
- **Every rpc is its own queue.** A slot's `poisoned` signal may arrive before
  or after that slot's transactions; a chain frame may arrive before or after the
  same legs on the per-leg rpc. Join on `(slot, signature)`; never assume
  cross-stream ordering.
- **A leg is on both intent rpcs.** `SubscribeSwapIntents` and
  `SubscribeSwapIntentChains` carry one set of legs under two groupings, each
  numbered on its own `publish_seq`. Subscribe to one; subscribing to both doubles
  the frames, not the information.
- **Signals degrade, they never disconnect** — same drop-oldest rule as
  `SubscribeEntries`. `publish_seq` is how you detect a gap; there is no other
  sequencing guarantee.
- **With no subscriber attached, nothing is computed.** Frames are not produced
  and thrown away, so the drop counters stay at zero — that is the idle state, not
  a fault.
- **A poisoned slot's transactions are withheld** from `TxSignal` — but only from the
  moment the poison is known. A slot can poison _after_ its prefix-decoded
  transactions already went out, and a published frame cannot be recalled. That is
  what the `poisoned` slot signal is for: treat it as a retraction covering every
  `TxSignal` you already hold for that slot.
- **A `TxSignal` can arrive before the entry frame carrying its bytes.** Speculative
  prefix decoding releases some transactions before their FEC set completes, and
  those reach this stream at that point rather than at batch time. Each transaction
  still appears exactly once — the pipeline drops an early-released transaction from
  the batch that follows — but `SubscribeEntries` always carries the full untouched
  payload, so the same transaction shows up there later. Do not treat a `TxSignal`
  with no matching entry frame yet as an error.

### What is not in it

Anything that needs execution: swap intents are their own stream (above),
lookup-table addresses ride in `account_keys_packed`, and venue activity in
`venue_touches`, but none of them says what the chain then did. Tips are a **floor**, not a total: a tip routed through a CPI is
invisible to a pre-execution decoder, like every other inner instruction.

## What is not in the stream

- **Transaction metadata**, in every form: logs, inner/CPI instructions, fees
  paid, compute units consumed, pre/post balances, and success or failure. None
  of it exists yet at the point deshred reads. If you need it, you need a
  post-execution source; the two are complements, not substitutes.
- **Resolved lookup-table addresses** — the indexes are forwarded, not expanded.
- **Decoded venue output on `SubscribeEntries`.** What you receive there is
  transactions, not swap intents — that wire is byte-identical to Jito's. The
  decoders' output is served on its own opt-in service — per leg on
  [`SubscribeSwapIntents`](#subscribeswapintents--stream-of-swapintentsignal),
  per instruction on
  [`SubscribeSwapIntentChains`](#subscribeswapintentchains--stream-of-swapintentchain)
  — with router legs (Jupiter v6, OKX, DFlow, Axiom) included whenever the hop's
  pool is already in the index. A routed hop on an unknown pool stands down
  rather than guessing; it is not a leg, and
  [`SubscribeRoutes`](#subscriberoutes--stream-of-routesignal) is where it is
  named.
- **Prometheus metrics.** The `/metrics` endpoint exists in the source but is
  behind a build feature that shipped binaries do not enable. Operational
  numbers come from the 10-second `dz::stats` log line instead.

Full list of structural limits, including the permanent ones:
[limits.md](limits.md).
