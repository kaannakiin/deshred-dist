# Honest limits

deshred's design rule is to **count what it cannot see instead of pretending
otherwise**. This page is the complete list of known structural limits. None of
them are bugs; all of them are measured.

## CPI blindness (permanent)

Shreds carry transactions as they were submitted: only pre-execution,
top-level instructions. A swap executed inside a cross-program invocation —
for example a venue swap invoked from inside an aggregator route — never
appears as a top-level instruction, so **no shred-level decoder can see it**;
this is a property of the data source shared by every project in this space,
not a deshred defect. Consequences:

- Venues whose flow is mostly CPI-wrapped have structurally low top-level
  coverage, and no future release can "fix" that.
- deshred reports such flow in explicit counters instead of inflating
  coverage; `verify`'s B matrix splits misses by cause, and CPI-wrapped misses
  dominate.
- The two router adapters (Jupiter v6, OKX DEX Router) recover _router-level_
  swap intent by decoding the router's own top-level instruction — that is a
  workaround at the router layer, not X-ray vision into CPIs.

## Pre-execution semantics

Everything deshred decodes is read **before execution**. That means:

- A decoded intent can belong to a transaction that later reverts. `verify`'s
  C matrix makes that population explicit.
- Some venue instructions state a **ceiling** rather than an exact trade (e.g.
  partial-fill modes); deshred carries the stated number and flags the mode
  instead of guessing the fill.
- Token-2022 transfer fees are modeled (net vs gross amounts are
  distinguished), but mint state can change on chain; the mint cache
  self-repairs with a staleness bound.

## Pre-execution signals

The opt-in `deshred.v1` streams carry these honest gaps:

- **A swap intent is a claim, not an outcome.** It is read from the instruction
  before execution: the transaction may revert, a partial-fill or price-bounded
  swap may fill less than stated, and a Token-2022 fee moves the vault by
  `amount_pool_side` rather than by the stated number. On the shipped slot 81 of
  196 intents sat on transactions that later failed.
- **Three venues need a pool index for their side.** Raydium AMM v4, Raydium
  CLMM's v1 `swap` and Meteora DAMM v1 name no mint in the instruction, so until
  this deployment's pool index holds the pool their intents ship with
  `direction_gap = POOL_UNKNOWN` and no mints. Measured over 560 mainnet slots
  that is 690 of 67,410 intents; every other venue resolves from its own bytes.
- **`amount_pool_side` is one-sided by design.** A transfer fee on the mint the
  swap mode does not pin leaves the stated amount exactly right, so no
  correction is applied there — applying one would manufacture the error.

- **An absent priority fee is not zero.** When a transaction states a compute-unit
  price but no limit, the runtime charges it against a default derived from the
  instruction mix _and_ the cluster's live feature set — which builtins have
  migrated to SBF. That is not a function of the transaction bytes, so deshred
  reports `LIMIT_UNSTATED` instead of guessing. Measured on the shipped capture:
  13 of 1243 fee-paying transactions, 1.05%.
- **Tip hits are a floor.** Only top-level System transfers to a known tip account
  are counted. A tip routed through a CPI is invisible, for the same permanent
  reason as everything else under CPI blindness above.
- **The durable-nonce witness is structural.** It says a transaction is _shaped_
  like a durable-nonce transaction — instruction 0 is a System
  `AdvanceNonceAccount` over a writable first account, which is the runtime's own
  test. It does not verify that the account currently holds initialized nonce
  state, and it does not classify who sent it: measured on the shipped capture,
  235 of 1537 non-vote transactions qualified, and they tipped at 16% — the same
  rate as the population at large, so this is not a bundle-sender detector. Note
  also that durable nonces may be deprecated upstream.
- **Per-slot FEC gap timing is not in the stream.** `batch_wait_us`, `decode_us`
  and `early_forward_us` are per-slot; `fec_gap_us` remains an aggregate in the
  10-second `dz::stats` log line only.

## Degraded mode (without an RPC endpoint)

Without `--rpc`/`SHRED_RPC` there is no leader schedule, so no FEC set can be
attributed to the leader that was scheduled for its slot. Those slots stay
`unverified`, and **an unverified slot is never served**: the gRPC stream
receives nothing and the decoders see nothing. `run` still starts and still
counts what arrived, so the health line reports a live feed while no subscriber
gets a frame — that combination points at the RPC endpoint, not the shred
source. Address-lookup-table misses cannot be filled either.

With an RPC endpoint, a bad leader signature poisons the slot (it is never
decoded), and a second differently-signed root for the same FEC set is treated
as equivocation and poisoned too.

### How hard deshred leans on that endpoint

Never harder than you allow. `SHRED_RPC` is your endpoint spending your quota,
so the request rate is capped rather than assumed: concurrent calls are bounded
by `SHRED_RPC_MAX_CONCURRENT` (default 16), the window halves itself whenever the
endpoint answers `429`/`502`/`503`/`504`, times out or drops the connection, and
misses are coalesced into batches of at most 100 keys within a
`SHRED_RPC_BATCH_WINDOW_MS` window. `--alt-journal` and `--mint-journal` remove
the cold start entirely on restart. Every knob is in
[env-reference.md](env-reference.md#rpc-budget), and the live window plus the
back-off counters appear on the `rpc health` log line every ten seconds.

⚠ Two things this does **not** do: it never retries a failed call (a miss is
re-requested by the next transaction that needs it, not by a retry loop), and it
never fans out to a second endpoint. One endpoint, capped.

## Platform

The low-latency claim lives on `x86_64-unknown-linux-gnu` only: `recvmmsg`,
busy-polling, and core pinning are Linux-only. macOS builds are correct but not
latency-competitive in `run`; `replay` and `verify` are fully supported there.
musl builds are not shipped until proven in CI.

## Measurement scope

- Published accuracy receipts come from captures taken at a single network
  vantage point. Multi-region generalization is unmeasured — treat it as
  unknown, not as implied.
- Throughput/latency numbers are withheld until they can be re-measured on
  reference hardware; this project does not ship numbers it cannot currently
  reproduce.
- Archives return no vote transactions; `verify` subtracts votes from the
  decoded side before any ratio, so coverage is honest and can never read
  above 100%.
