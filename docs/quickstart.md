# Quickstart

The fastest working setup is **unicast via jito-shredstream-proxy** — no
DoubleZero seat required.

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

## 2. Point a shredstream proxy at it

Run [`jito-shredstream-proxy`](https://github.com/jito-labs/shredstream-proxy)
wherever you already receive shreds, and add this host to its
`--dest-ip-ports`:

```sh
--dest-ip-ports <deshred-host>:7733
```

`7733` is deshred's default `DESHRED_PORT`; change either side to match.

## 3. Run

```sh
# DESHRED_GROUPS left empty -> unicast mode. There is no separate unicast flag.
./deshred run --rpc https://your-rpc-endpoint
```

- `--rpc` (or `DESHRED_RPC`) is optional but strongly recommended: without it,
  leader signatures cannot be checked and everything stays `unverified` — see
  [limits.md](limits.md#degraded-mode-without-an-rpc-endpoint).
- The decoded feed is served as gRPC on `127.0.0.1:9900` — see
  [grpc.md](grpc.md).
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

If you have a DoubleZero seat, see [multicast.md](multicast.md).
