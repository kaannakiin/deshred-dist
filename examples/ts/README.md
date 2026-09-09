# TypeScript: subscribe to `deshred.v1.SignalStream`

The shortest path from nothing to frames: `@grpc/proto-loader` reads the
shipped `.proto` files at startup, so there is **no codegen step and no
`protoc`** anywhere in this example.

## Run

```sh
npm ci
npm start                       # defaults to 127.0.0.1:9900
npm start -- 127.0.0.1:9900
```

The stream is opt-in on the server side, so start deshred with
`SHRED_GRPC_SIGNALS=true`. If you forget, this example prints the server's own
refusal rather than hanging:

```
refused: SubscribeTxSignals is not published by this process; start it with SHRED_GRPC_SIGNALS=true
```

Otherwise, one line per transaction:

```
slot 351234567 tx 42 keys 27 fee 12500 lamports 4Nd8kQ2vXrPmSaTb…
```

## Why this example does not decode `entries`

`@solana/web3.js`'s `VersionedTransaction.deserialize` consumes a prefix of the
buffer and does not report how many bytes it used, and an entry's transactions
carry no length prefix between them — so there is no supported way to find
where the next transaction starts. You would have to walk the wire format
yourself; [../go/](../go/) shows that walk, and it ports to any language.

`SignalStream` is usually the better answer for a JS consumer anyway: it hands
over the signature, the transaction index and the **ALT-resolved**
`account_keys_packed` — which `SubscribeEntries` structurally cannot, because
resolving a lookup table needs chain state that is not in the shred.

## Notes

- `longs: String` keeps `uint64` slots exact; the default `Long` objects and
  plain JS numbers both lose precision above 2^53.
- `enums: String` makes gap fields readable (`PRIORITY_FEE_GAP_LIMIT_UNSTATED`
  rather than `3`) — an absent priority fee is never simply zero, see
  [../../docs/data-model.md](../../docs/data-model.md).
- `signature` arrives as raw bytes, not base58. Encode it yourself to join
  against an explorer.
