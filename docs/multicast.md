# Multicast ingest

deshred can join shred multicast groups directly. It ships **no
provider-specific group defaults** — your provider assigns your groups and the
interface they arrive on; enter what they give you. DoubleZero is the deployed
example of such a fabric, and the notes below name it where its conventions are
what you will actually see.

Other ways to feed deshred — a `jito-shredstream-proxy` relay, your own node, a
capture file — are in [shred-feed.md](shred-feed.md).

## Configuration

```sh
export SHRED_IFACE=<interface carrying the feed>   # e.g. doublezero1 on a DoubleZero fabric
export SHRED_GROUPS=<group1>,<group2>              # comma-separated, from your provider
export SHRED_PORT=7733                             # default
./deshred run --rpc https://your-rpc-endpoint
```

Setting `SHRED_GROUPS` to one or more addresses switches ingest to multicast
mode; leaving it empty keeps unicast mode. There is no separate mode flag.

`SHRED_IFACE` has no default and is **required** in multicast mode: a join with
no interface would land on `0.0.0.0` and silently miss the feed, so deshred
refuses at boot instead. Unicast mode never reads it.

## Address validation

deshred validates that each group is a real multicast address (`224.0.0.0/4`)
and refuses anything outside that space. It additionally **warns** — but does
not refuse — when a group falls outside `233.84.178.0/24`, the publicly
documented DoubleZero shred multicast range. On a DoubleZero fabric a warning
usually means a typo or a stale assignment; on any other fabric it is expected
and can be ignored.

## Troubleshooting a silent feed

1. Confirm the interface name: `ip link`. On a DoubleZero fabric these are
   typically named `doublezero*`. Set `SHRED_IFACE` accordingly.
2. Confirm the group and port with your provider — both must match exactly.
3. Check that the host actually joined: `ip maddr show dev <iface>` should list
   your group after startup. On point-to-point tunnels the join can fail while
   datagrams still arrive — deshred warns rather than exiting.
4. Check firewall/rp_filter settings; multicast UDP is commonly dropped by
   default policies.
5. The 10-second stats log line shows received datagram counts — zero datagrams
   with a successful join points at the network path, not at deshred.
6. `./deshred probe --discover` (Linux) inventories every multicast group and
   port actually arriving on the interface, joining nothing. It is the fastest
   way to tell "wrong group" from "nothing is being sent".

More in [troubleshooting.md](troubleshooting.md).
