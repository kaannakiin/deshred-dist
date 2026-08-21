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
- The two router adapters (Jupiter v6, OKX DEX Router) recover *router-level*
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

## Degraded mode (without an RPC endpoint)

Without `--rpc`/`DESHRED_RPC` there is no leader schedule, so FEC-set leader
signatures cannot be checked: slots are admitted but stay `unverified`, and
address-lookup-table resolution misses cannot be filled. `run` starts anyway —
deliberately — but a degraded feed should not be consumed for anything that
matters. With an RPC endpoint, a bad leader signature poisons the slot (it is
never decoded), and a second differently-signed root for the same FEC set is
treated as equivocation and poisoned too.

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
