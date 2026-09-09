# Go: decode `SubscribeEntries` with no Solana dependency

`Entry.entries` is a bincode `Vec<Entry>`. The outer framing is three fixed
reads, but the transactions inside an entry have **no length prefix between
them** — the only way to find the next one is to walk the current one. This
example does exactly that, in ~150 lines, with no Solana library at all.

## Run

```sh
./gen.sh                 # protoc → gen/shredstream, then `go mod tidy`
go run . 127.0.0.1:9900
```

`gen.sh` needs `protoc` plus the two Go plugins:

```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Output, one line per frame:

```
slot            5  txs    1  legacy   1  v0   0  first 99eUso3aSbE9tqGSTXzo3TLfKb9…
```

## The part worth reading

`cursor.transaction()` consumes one wire transaction: compact-u16 signature
count, 64 bytes each, then the message — the `0x80`-flagged version byte if
present, the three-byte header, the account-key vector, the blockhash, the
instruction vector (each with its own account-index and data vectors), and for
v0 the address-lookup-table vector.

`decodeEntries` ends with the check that makes the walk trustworthy:

```go
if c.at != len(payload) {
    return out, fmt.Errorf("walked %d of %d bytes — a transaction boundary was misread", ...)
}
```

A correct walk lands **exactly** on the end of the payload. Any boundary
misread shows up as leftover or overrun bytes, so the example reports a
diagnostic instead of printing plausible nonsense. Keep that check if you port
this.

The walk was verified against a committed 229-byte entry payload (fully
consumed, one legacy transaction) and against a synthesized frame carrying both
a legacy and a v0 transaction with lookup tables (518 bytes, fully consumed).

## Why no `option go_package`

The Jito protos are vendored byte-for-byte and carry no Go package option, so
`gen.sh` supplies it with `M<file>=` mappings rather than editing a file whose
bytes are pinned. See [../../proto/jito-shredstream/PIN.md](../../proto/jito-shredstream/PIN.md).
