# `deshred.v1`

deshred's own signal surface: three files, one protobuf package.

| File | Service | Carries |
| --- | --- | --- |
| `signals.proto` | `SignalStream` | `TxSignal` (per-transaction pre-execution facts) and `SlotSignal` (per-slot observations) |
| `intents.proto` | `SwapIntentStream` | `SwapIntentSignal` per venue swap leg, and `SwapIntentChain` per instruction |
| `routes.proto` | `RouteStream` | `RouteSignal` — every hop of a router's plan, claimed or not |

`intents.proto` imports `signals.proto`; `routes.proto` imports both. Nothing
here is vendored — see [../jito-shredstream/](../jito-shredstream/) for the
Jito-compatible entry stream, which is a separate package with a separate
license.

## The compatibility promise

The schema is **additive within `deshred.v1`**:

- A tag number is never renumbered, retyped or repurposed. A field's number and
  type, once shipped, are fixed.
- A tag is `reserved` only when a field is actually removed.
- Every enum's `0` value is an `_UNSPECIFIED` that the server never sends. An
  older client reading a value your build does not know reads it as
  `_UNSPECIFIED` and fails closed rather than mistaking it for a real variant.
- A breaking change gets a **new package**, not an edit to this one.

So a client generated from these files keeps working across deshred releases,
and unknown fields are ignored by protobuf as usual.

**One documented exception.** `RouteSignal.mode` *does* send
`SWAP_MODE_UNSPECIFIED` — a route plan whose mode the decoder could not
determine says so there, and `plan_stand_down` carries the reason. Every other
enum on every other field follows the never-sent rule.

## Reading the field comments

The comments in these files are the contract, not decoration: they say what a
field means when it is absent, which `*_Gap` enum explains a missing value, and
which numbers were measured against confirmed mainnet blocks rather than
assumed. Read them before treating any field as a number you can trade on.

Field-by-field semantics: [../../docs/data-model.md](../../docs/data-model.md).
Tag numbers and enum values in one table:
[../../docs/proto-reference.md](../../docs/proto-reference.md).
