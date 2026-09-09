# Where deshred sits, and how to feed it

deshred is not a feed provider. It is a decoder that reads raw UDP datagrams off
a socket you point at it. Any source that hands it Agave shred bytes works —
what follows is where those bytes come from on the Solana network, and the four
ways to get them into the process.

## Where the bytes come from

A leader packs transactions into **entries**, splits each batch into **shreds**
(fixed-size UDP payloads: data shreds carrying the entries, coding shreds
carrying Reed–Solomon parity), groups them into **FEC sets**, and broadcasts
them down the **Turbine** tree. Every validator on that tree relays the same
datagrams to its children. This is the earliest point at which a transaction
exists outside the leader, and it is the stage deshred reads.

Three consequences, all of them load-bearing:

- **It is pre-execution.** A shred carries the transaction as submitted — top-level
  instructions, account keys, signatures. No logs, no balances, no status, and
  nothing that happened inside a CPI. That limit is permanent and deshred counts
  what it cannot see rather than hiding it:
  [limits.md](limits.md#cpi-blindness-permanent).
- **It is attributable.** Each FEC set's Merkle root is signed by the slot's
  leader. deshred checks that signature against a leader schedule derived from
  your RPC endpoint before any transaction from that slot is served. A bad
  signature poisons the slot; a second, differently-signed root for the same
  slot is treated as equivocation and also poisons it. Nothing poisoned is ever
  forwarded.
- **It is incomplete until it is not.** A FEC set arrives out of order and
  sometimes short; missing shreds are recovered from the coding shreds using
  Agave's own Reed–Solomon implementation, not a reimplementation.

## Getting the datagrams

deshred never dials anything for its transaction feed. It binds a UDP socket and
reads. `SHRED_GROUPS` is the only mode switch: empty means a plain unicast bind,
set means it joins those multicast groups. There is no separate mode flag.

| Source                                                                                                       | Mode      | `SHRED_GROUPS`         | `SHRED_IFACE` |
| ------------------------------------------------------------------------------------------------------------ | --------- | ---------------------- | ------------- |
| [`jito-shredstream-proxy`](https://github.com/jito-labs/shredstream-proxy) relay — the fastest working setup | unicast   | empty                  | unused        |
| Your own validator or node already on the Turbine tree, forwarding raw shred UDP                             | unicast   | empty                  | unused        |
| A multicast shred fabric (DoubleZero is the deployed example)                                                | multicast | your provider's groups | required      |
| A capture file — no live feed at all                                                                         | n/a       | n/a                    | n/a           |

**Unicast via a proxy.** Run the proxy wherever you already receive shreds and
add this host to its `--dest-ip-ports`:

```sh
--dest-ip-ports <deshred-host>:7733
```

`7733` is deshred's default `SHRED_PORT`; change either side to match.

**Unicast from your own node.** Anything that copies the raw shred datagrams to
`SHRED_PORT` on this host works — the payload deshred wants is the unframed
Agave shred, exactly as it came off the wire. No framing, no envelope, no
length prefix.

**Multicast.** Set `SHRED_IFACE` and `SHRED_GROUPS` to what your provider
assigned. deshred ships no provider-specific group defaults. Details, address
validation and a silent-feed checklist: [multicast.md](multicast.md).

**A capture file.** `deshred replay --capture <file> --port 7733` decodes
without a feed at all; it reads classic pcap and deshred's own DZCAP3 format,
telling them apart by the file's magic bytes. `deshred capture` records DZCAP3.

The payload is identical raw Agave shred bytes in every live case above. The two
socket modes — multicast join versus plain bind, derived from whether
`SHRED_GROUPS` is empty — are the only place the difference exists anywhere in
the pipeline. A proxy's own heartbeat datagrams are recognised and discarded.

## What else it needs

One side channel, and it is not optional in practice: `SHRED_RPC`, an ordinary
Solana JSON-RPC endpoint. It supplies the leader schedule and shred version at
boot, and address-lookup-table and Token-2022 mint accounts on demand. Without
it no slot can be attributed to a leader, so every slot stays `unverified` and
nothing is served — see
[limits.md](limits.md#degraded-mode-without-an-rpc-endpoint) and
[env-reference.md](env-reference.md#rpc-budget) for the rate-limit knobs, since
this spends _your_ endpoint's credits.

It carries no transaction data. deshred never signs anything and has no
execution path.

## A note on names

`dz::*` log targets, the `.dzcap` capture format and the `dzl1.` license-key
prefix are internal names from this project's early history. None of them
implies a DoubleZero dependency, and none of the code paths behind them is
provider-specific.

Next: [quickstart.md](quickstart.md) to get a feed running,
[grpc.md](grpc.md) to consume it, [data-model.md](data-model.md) for the
contract.
