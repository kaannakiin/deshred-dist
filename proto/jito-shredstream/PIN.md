# Vendored: jito-labs/mev-protos

- Source: https://github.com/jito-labs/mev-protos
- Pinned commit: `46ead86a13a55a0ef2c139db96a8ee93bf7505e3` (2025-07-23;
  `shredstream.proto` itself last changed in `9dbd8013b57442b1511a57be7e469a43b37f151f`,
  2025-04-03)
- Fetched: 2026-08-20
- License: Apache-2.0, Copyright (c) 2025 Jito Labs. The `LICENSE` file in this
  directory was copied from the same commit. That commit carries no `NOTICE`
  file.
- Files: `shredstream.proto`, `shared.proto` — **vendored verbatim, zero
  modifications.** `shared.proto` is here because `shredstream.proto` imports it
  (`shared.Socket`).

`google/protobuf/timestamp.proto` is deliberately **not** vendored: it is a
protobuf well-known type that every toolchain resolves on its own. See
[../README.md](../README.md) if your `protoc` cannot find it.

These two files are byte-pinned in deshred's own test suite, so a silent
truncation or a fork cannot reach a release. The copies in this repository are
byte-identical to the ones the shipped binary was compiled from — that is also
enforced by a test, and re-checked before every tag.
