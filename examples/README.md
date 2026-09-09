# Example clients

Three subscribers, one per language, each pointed at the schemas in
[../proto/](../proto/). Pick by what you need out of the stream.

| Example | Stream | Decodes `entries`? | Needs |
| --- | --- | --- | --- |
| [rust/](rust/) | `shredstream.ShredstreamProxy/SubscribeEntries` | **Yes** — one `bincode` call gives you `Vec<Entry>` and every transaction | Rust + `protoc` |
| [go/](go/) | same | **Yes** — via a portable wire walk, no Solana library | Go + `protoc` + two Go plugins |
| [ts/](ts/) | `deshred.v1.SignalStream/SubscribeTxSignals` | No, by design — see its README | Node only, no codegen |

## Why the difference

`Entry.entries` is a bincode `Vec<Entry>`, forwarded byte-for-byte from the
shreds. Rust deserializes it in one call. Everywhere else the obstacle is that
transactions inside an entry have no length prefix between them, so a consumer
must walk each transaction to find the next — the Go example does this
explicitly, with a self-check that fails loudly if a boundary is misread.

If you only want facts *about* transactions — signature, priority fee, tips,
durable nonce, resolved account keys, provenance — take `deshred.v1.SignalStream`
instead and skip entry decoding entirely. That is what the TypeScript example
does, and it is the shortest path in any language.

## Running them without a live feed

Any server speaking the pinned proto works, so you can develop against
something local before you have a shred feed. Point the example at whatever
that is; the entry decoders fail loudly on a payload that is not a bincode
`Vec<Entry>` rather than printing nonsense.

## Scope

These are subscribers, not libraries: no reconnect policy, no backpressure
handling, no metrics. Read [../docs/data-model.md](../docs/data-model.md#delivery-and-backpressure)
before running one against something that matters — in particular, each rpc is
its own queue with a drop-oldest ring, and `publish_seq` is how you notice a
dropped frame.
