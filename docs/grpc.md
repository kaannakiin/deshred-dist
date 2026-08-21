# Consuming the gRPC feed

`deshred run` serves reconstructed entries over gRPC using the same wire
contract as Jito's shredstream proxy: service `shredstream.ShredstreamProxy`,
method `SubscribeEntries`, from
[jito-labs/mev-protos](https://github.com/jito-labs/mev-protos) `shredstream.proto`
(vendored at a pinned commit, Apache-2.0). An existing Jito shredstream
consumer connects with no client-side changes — same proto, same stream shape.

## Endpoint

| Setting | Default | Meaning |
| --- | --- | --- |
| `DESHRED_GRPC_LISTEN_ADDR` | `127.0.0.1:9900` | TCP listen address. |
| `DESHRED_GRPC_UDS` | *(unset)* | Serve on a Unix domain socket at this path instead. |

The default binds loopback only — put your own proxy/TLS in front if you need
to expose it beyond the host.

## Smoke test

```sh
grpcurl -plaintext 127.0.0.1:9900 list
# shredstream.ShredstreamProxy

grpcurl -plaintext 127.0.0.1:9900 shredstream.ShredstreamProxy/SubscribeEntries
```

Each stream message carries a slot number and serialized ledger entries; the
entries deserialize with the standard Solana entry format, exactly as they do
from a Jito shredstream endpoint.

## Trust semantics

Entries are published only from slots that passed admission. When no RPC
endpoint is configured, leader signatures cannot be checked and slots are
`unverified` — see [limits.md](limits.md#degraded-mode-without-an-rpc-endpoint)
before consuming a degraded feed for anything that matters.
