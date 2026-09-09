# Quickstart

The fastest working setup is **unicast via jito-shredstream-proxy**. Every other
way to get a shred feed — your own node, a multicast fabric, a capture file — is
in [shred-feed.md](shred-feed.md).

## 1. Install

Download the tarball for your platform from
[Releases](../../../releases), verify the checksum, unpack:

```sh
sha256sum -c <(grep x86_64-unknown-linux-gnu checksums.txt)
tar xzf deshred-<version>-x86_64-unknown-linux-gnu.tar.gz
cd deshred-<version>-x86_64-unknown-linux-gnu
./deshred --version
```

The tarball contains the `deshred` binary, `THIRD-PARTY-LICENSES.txt`, and
`EULA.txt`.

To consume the gRPC stream you also want the schemas — `proto-kit-<version>.tar.gz`
from the same release, or the [`proto/`](../proto/) directory of this
repository. This server serves no gRPC reflection, so your client (grpcurl
included) needs the files:

```sh
tar xzf proto-kit-<version>.tar.gz          # unpacks proto/
```

## 2. Point a shredstream proxy at it

Run [`jito-shredstream-proxy`](https://github.com/jito-labs/shredstream-proxy)
wherever you already receive shreds, and add this host to its
`--dest-ip-ports`:

```sh
--dest-ip-ports <deshred-host>:7733
```

`7733` is deshred's default `SHRED_PORT`; change either side to match.

## 3. Run

```sh
# SHRED_GROUPS left empty -> unicast mode. There is no separate unicast flag.
export SHRED_LICENSE=<your-key>     # or: ./deshred run --license <your-key> ...
./deshred run --rpc https://your-rpc-endpoint
```

- `run` is the only command that needs a license key (see
  [env-reference.md](env-reference.md#license)); `replay`, `verify` and
  `capture` work without one.
- `--rpc` (or `SHRED_RPC`) is optional but strongly recommended: without it,
  leader signatures cannot be checked and everything stays `unverified` — see
  [limits.md](limits.md#degraded-mode-without-an-rpc-endpoint).
- The decoded feed is served as gRPC on `127.0.0.1:9900` — see
  [grpc.md](grpc.md). First subscribe:

  ```sh
  grpcurl -plaintext \
    -import-path proto/jito-shredstream -proto shredstream.proto \
    127.0.0.1:9900 shredstream.ShredstreamProxy/SubscribeEntries
  ```

  Runnable Rust, Go and TypeScript clients: [../examples/](../examples/).

- A stats line is logged every 10 seconds; a healthy feed shows datagrams and
  completed slots climbing.

## 4. Prove it (optional but recommended)

Before trusting the feed, run the offline verification fixture shipped with
every release — see [verify.md](verify.md). It takes under a minute and needs
no network.

## Offline decoding instead of a live feed

If you have a capture file, no feed is needed at all:

```sh
./deshred replay --capture your-capture.dzcap --port 7733
```

`replay` accepts both classic pcap and deshred's own DZCAP3 format, telling
them apart by the file's magic bytes. `deshred capture` records DZCAP3.

## Multicast instead of unicast

If your feed arrives on a multicast fabric, set `SHRED_IFACE` and `SHRED_GROUPS`
instead of leaving `SHRED_GROUPS` empty — see [multicast.md](multicast.md).
