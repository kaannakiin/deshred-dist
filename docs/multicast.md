# Multicast (DoubleZero)

deshred can join DoubleZero shred multicast groups directly. It ships **no
provider-specific group defaults** — your provider assigns your groups; enter
what they give you.

## Configuration

```sh
export DESHRED_IFACE=<your-doublezero-interface>   # default: doublezero1
export DESHRED_GROUPS=<group1>,<group2>            # comma-separated, from your provider
export DESHRED_PORT=7733                           # default
./deshred run --rpc https://your-rpc-endpoint
```

Setting `DESHRED_GROUPS` to one or more addresses switches ingest to multicast
mode; leaving it empty keeps unicast mode. There is no separate mode flag.

## Address validation

deshred validates that each group is a real multicast address (`224.0.0.0/4`)
and refuses anything outside that space. It additionally **warns** — but does
not refuse — when a group falls outside `233.84.178.0/24`, the publicly
documented DoubleZero shred multicast range. A warning usually means a typo or
a stale assignment; double-check with your provider.

## Troubleshooting a silent feed

1. Confirm the interface name: `ip link` — DoubleZero interfaces are typically
   named `doublezero*`. Set `DESHRED_IFACE` accordingly.
2. Confirm the group and port with your provider — both must match exactly.
3. Check that the host actually joined: `ip maddr show dev <iface>` should list
   your group after startup.
4. Check firewall/rp_filter settings; multicast UDP is commonly dropped by
   default policies.
5. The 10-second stats log line shows received datagram counts — zero datagrams
   with a successful join points at the network path, not at deshred.

More in [troubleshooting.md](troubleshooting.md).
