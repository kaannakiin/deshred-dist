<!--
Şablon — her tag için release-notes/vX.Y.Z.md olarak kopyalanıp doldurulur.
release.yml guard-version job'u dosyanın tag'den ÖNCE var olmasını zorlar;
publish-dist bunu gh release create --notes-file ile dist repoya taşır.
<...> yer tutucularının hepsi doldurulmadan tag atılmaz.
-->

## Pins

- `solana-ledger` = `2.3.13` — the shred wire format and FEC recovery this
  binary was built and verified against.
- Vendored `jito-labs/mev-protos` commit: `46ead86a13a55a0ef2c139db96a8ee93bf7505e3` (Apache-2.0).

## Prove this binary

Download `verify-kit-<vX.Y.Z>.tar.gz` from this release and run:

```sh
deshred verify --capture slice.dzcap --truth truth.json --alt-journal alt-journal.jsonl \
  --pools pools.json --expect expected.json
```

Expected: `VERIFY GREEN — matrices match expected.json`. Every artefact in this
release passed this exact check in the release pipeline before publishing — an
artefact that fails it is never published. File hashes are in the kit's
`manifest.json` and below:

| File | SHA-256 |
| --- | --- |
| slice.dzcap | `<sha256>` |
| truth.json | `<sha256>` |
| pools.json | `<sha256>` |
| alt-journal.jsonl | `<sha256>` |
| expected.json | `<sha256>` |

The fixture slot is mainnet 440061516: 2,213 transactions reconstructed
pre-execution, 1,537 non-vote, matching a public archive's confirmed set
exactly (zero difference either direction).

## Hardening

- Build profile: `lto=fat`, `codegen-units=1`, `opt-level=3`, `strip=symbols`,
  plus `--remap-path-prefix` over the checkout, the cargo registry, and the
  build home — no build-machine paths ship in the binary.
- `panic = "abort"`: **not enabled in this release.** Unwind is kept so a
  user-reported panic still carries a readable message; strip+remap already
  remove symbols and source paths. Revisited per release, on purpose.
- `strings` leak scan: clean — no private repository path, fork URL, internal
  hostname, username, or configuration key name appears in the shipped binary.

## Changes

<!-- Kullanıcıya görünen değişiklikler; iç refactor listelenmez. -->

- <...>

## Known limits

Unchanged and documented: [docs/limits.md](../docs/limits.md) — CPI-wrapped
swaps are structurally invisible at the shred layer; running without
`--rpc`/`DESHRED_RPC` degrades to `unverified`; macOS `run` is not
latency-competitive; throughput/latency numbers are withheld pending
recalibration.

## Checksums

`checksums.txt` in this release lists the SHA-256 of every platform tarball.
