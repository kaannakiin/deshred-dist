# Protobuf schemas

Everything deshred serves over gRPC, as the exact `.proto` files the shipped
binary was compiled from. Generate a client from these; there is nothing else to
install.

```
proto/
  jito-shredstream/   shredstream.proto, shared.proto, LICENSE, PIN.md   (vendored, Apache-2.0)
  deshred-v1/         signals.proto, intents.proto, routes.proto, README.md   (deshred's own)
```

Two directories because they are two packages with two licenses:
`shredstream` is [Jito's](jito-shredstream/PIN.md), vendored verbatim;
`deshred.v1` is [ours](deshred-v1/README.md) and carries an additive-only
compatibility promise.

**This server does not serve the gRPC reflection API.** `grpcurl … list`
against a running deshred fails with `server does not support the reflection
API`; that is expected, not a misconfiguration. Point your tooling at these
files instead — which also means `list` and `describe` work with no server
running at all, and tell you the schema you actually hold.

## Include paths

Imports inside these files are flat — `shredstream.proto` does
`import "shared.proto"`, `intents.proto` and `routes.proto` do
`import "signals.proto"` — so the include root is the **directory**, and file
names on the command line are spelled relative to a root.

```sh
# the entry stream alone
protoc -I proto/jito-shredstream --descriptor_set_out=/dev/null shredstream.proto

# the deshred.v1 signal streams alone
protoc -I proto/deshred-v1 --descriptor_set_out=/dev/null \
       signals.proto intents.proto routes.proto

# everything in one invocation: pass both roots
protoc -I proto/jito-shredstream -I proto/deshred-v1 --descriptor_set_out=/dev/null \
       shredstream.proto signals.proto intents.proto routes.proto
```

Two failures worth recognising (both verified against `libprotoc 35.1`):

- `File does not reside within any path specified using --proto_path (or -I).`
  — you named a file that no `-I` root covers. Add the root that contains it.
- `Could not make proto path relative: shredstream.proto: No such file or
directory` — you named a file relative to a root you did not pass. Add
  `-I proto/jito-shredstream`.

## `google/protobuf/timestamp.proto`

`shredstream.proto` and `shared.proto` import it, but only for `TraceShred` and
`shared.Header` — neither is on the `SubscribeEntries` path. It still has to
_resolve_, and it is not vendored here because every toolchain supplies it:
`protoc` from its own bundled `include/` directory, `grpcurl` and `buf`
compiled in, `@grpc/proto-loader` from protobuf.js, `grpcio-tools` from its
wheel. The one way to break this is a hand-unpacked `protoc` whose `include/`
directory was discarded — reinstall it from a package manager and the import
resolves.

## Generating a client

**grpcurl** — no codegen. Browse the schema offline:

```sh
grpcurl -import-path proto/deshred-v1 \
        -proto signals.proto -proto intents.proto -proto routes.proto list
grpcurl -import-path proto/deshred-v1 -proto intents.proto \
        describe deshred.v1.SwapIntentSignal
```

…and subscribe to a running deshred with the same pair of flags:

```sh
grpcurl -plaintext \
  -import-path proto/jito-shredstream -proto shredstream.proto \
  127.0.0.1:9900 shredstream.ShredstreamProxy/SubscribeEntries
```

**Go** — `protoc` plus the two Go plugins:

```sh
protoc -I proto/jito-shredstream -I proto/deshred-v1 \
       --go_out=gen --go_opt=paths=source_relative \
       --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
       shredstream.proto shared.proto signals.proto intents.proto routes.proto
```

**Rust / tonic** — in `build.rs`:

```rust
tonic_prost_build::configure().compile_protos(
    &["shredstream.proto", "signals.proto", "intents.proto", "routes.proto"],
    &["proto/jito-shredstream", "proto/deshred-v1"],
)?;
```

All three `deshred.v1` files share one package, so they land in **one**
generated module.

**TypeScript / Node** — no codegen at all; `@grpc/proto-loader` reads the files
at startup:

```ts
const def = protoLoader.loadSync(
  ["signals.proto", "intents.proto", "routes.proto"],
  { includeDirs: ["proto/deshred-v1"], longs: String, defaults: true },
);
```

**Python** — `python -m grpc_tools.protoc` takes the same `-I` roots as
`protoc` above, plus `--python_out` / `--grpc_python_out`.

Runnable versions of the first three: [../examples/](../examples/).

## Which stream do you want

| You want                                                     | Service                                         | Notes                                                                                                                       |
| ------------------------------------------------------------ | ----------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| Raw pre-execution transactions, Jito-compatible              | `shredstream.ShredstreamProxy/SubscribeEntries` | Always on. `Entry.entries` is a bincode `Vec<Entry>` — see [../docs/data-model.md](../docs/data-model.md#unpacking-entries) |
| Priority fee, tips, nonce, provenance, resolved account keys | `deshred.v1.SignalStream`                       | Needs `SHRED_GRPC_SIGNALS=true`                                                                                             |
| Decoded venue swap legs                                      | `deshred.v1.SwapIntentStream`                   | Needs `SHRED_GRPC_SIGNALS=true` **and** `SHRED_DECODE_INTENTS=true`                                                         |
| Router route plans, hop by hop                               | `deshred.v1.RouteStream`                        | Same two knobs                                                                                                              |

Every request message is empty: there are no server-side filters, every
subscriber gets the same stream, and filtering is client-side. Field-by-field
semantics live in [../docs/data-model.md](../docs/data-model.md); tag numbers
and enum values in [../docs/proto-reference.md](../docs/proto-reference.md).
