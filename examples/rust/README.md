# Rust: decode `SubscribeEntries`

Full-fidelity entry decode in one call — `bincode` hands you
`Vec<solana_entry::entry::Entry>` and every transaction inside it.

## Run

```sh
cargo run                                  # defaults to http://127.0.0.1:9900
cargo run -- http://127.0.0.1:9900
```

Needs `protoc` on PATH (the build script generates the client from
[`../../proto/jito-shredstream`](../../proto/jito-shredstream)).

Output, one line per frame:

```
slot            5  entries    1  txs    1  first 99eUso3aSbE9tqGSTXzo3TLfKb9RkMTURrHKQ1K7Zh3B…
```

## What it shows

- The `bincode` options that match the wire exactly: fixint encoding, trailing
  bytes allowed, limited to the frame (`src/main.rs`). Anything else silently
  mis-parses.
- Two proto modules, not one: the generated `shredstream.rs` refers to
  `super::shared::Socket`, so `shared.rs` has to be a sibling module — and
  `prost-types` has to be a dependency for `TraceShred.created_at`, even though
  neither type is on the `SubscribeEntries` path.
- `signatures.first()` is the transaction's id, and the join key against
  `deshred.v1.TxSignal.signature`.

## No feed handy?

Any server speaking the pinned proto works. Frames whose payload is not a
bincode `Vec<Entry>` fail the decode loudly rather than printing nonsense.

## Lighter alternative

`solana-entry` pulls a chunk of the SDK. If you only want the transaction
bytes, declare the frame's shape yourself — `#[derive(Deserialize)] struct Entry
{ num_hashes: u64, hash: [u8; 32], transactions: Vec<VersionedTransaction> }`
over `solana-transaction` — and keep the same `bincode` options.
