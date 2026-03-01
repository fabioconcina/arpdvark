# arpdvark

A minimal, fast terminal-based network inventory tool. Scans your local network using ARP, identifies connected devices, resolves hostnames and MAC vendors, and displays results in a full-screen auto-refreshing TUI.

```
╭─────────────────────────────────────────────────────────────────────────────────╮
│ arpdvark  •  interface: eth0  •  subnet: 192.168.1.0/24                         │
│ IP Address       MAC Address           Hostname                    Vendor        │
│ 192.168.1.1      30:1f:48:10:f3:04     router.home                zte corp.     │
│ 192.168.1.2      a8:1d:16:31:a6:4f     laptop.home                AzureWave     │
│ 192.168.1.5      98:e2:55:7f:8a:48                                Nintendo       │
│ 192.168.1.112    34:af:b3:82:16:95     echo.home                  Amazon Tech.  │
│ 192.168.1.136    ea:03:65:53:c9:62                                Local/Random. │
│ 5 device(s)  •  last scan: 2s ago  •  r: rescan  q: quit                        │
╰─────────────────────────────────────────────────────────────────────────────────╯
```

**Supported platforms:** Linux (amd64, arm64)

## Features

- **ARP scanning** — discovers all active hosts on the local subnet using raw ARP packets
- **Hostname resolution** — tries three methods in order: system DNS, gateway DNS (dnsmasq/home routers), mDNS unicast (Bonjour/Avahi)
- **MAC vendor lookup** — identifies hardware vendors from the embedded [IEEE OUI registry](https://standards-oui.ieee.org/oui/oui.txt) (~39k entries)
- **Locally administered MAC detection** — flags randomized, VM-assigned, or manually set MACs as `Local/Randomized` (these have no OUI entry by design)
- **Full-screen TUI** — fills the terminal, resizes dynamically, auto-refreshes on a configurable interval
- **Persistent device table** — devices seen in previous scans remain visible; `LastSeen` is updated on each round

## Installation

**From source:**

```sh
git clone https://github.com/fabioconcina/arpdvark
cd arpdvark
make build-all          # produces dist/arpdvark-linux-amd64 and dist/arpdvark-linux-arm64
```

Copy the appropriate binary to your Linux machine and place it in your `PATH`.

## Usage

```
Usage: arpdvark [options]

Options:
  -i <interface>    Network interface to scan (default: auto-detect)
  -t <seconds>      Refresh interval in seconds (default: 10)
  -large            Allow scanning subnets larger than /16
  -h                Show help
```

**Examples:**

```sh
# Auto-detect interface, refresh every 10s
sudo arpdvark

# Scan a specific interface every 5 seconds
sudo arpdvark -i eth0 -t 5

# Scan a large subnet (up to /8)
sudo arpdvark -i eth0 -large
```

**Key bindings:**

| Key | Action |
|-----|--------|
| `q` / `ctrl+c` | Quit |
| `r` | Force immediate rescan |
| `↑` / `↓` | Navigate table rows |

## Permissions

arpdvark uses raw ARP sockets (`AF_PACKET`) and requires elevated privileges. Run with `sudo`, or grant `CAP_NET_RAW` to avoid it:

```sh
sudo setcap cap_net_raw+ep ./arpdvark
./arpdvark
```

## Hostname resolution

Hostnames are resolved concurrently for all discovered devices after each ARP sweep. Three methods are tried in order:

1. **System resolver** — uses `/etc/resolv.conf`. Works if your DNS server (e.g. Pi-hole with DHCP enabled) maintains PTR records.
2. **Gateway DNS** — queries the first host in the subnet (typically the router) directly on port 53. Home routers running dnsmasq create PTR records for DHCP leases automatically.
3. **mDNS unicast** — sends a PTR query to the device's port 5353. Works for Apple devices (Bonjour) and Linux hosts running Avahi.

If all three fail (e.g. the device has a randomized MAC and no mDNS), the hostname column is left empty.

## Vendor database

The bundled OUI database is embedded at build time from `vendor_db/oui.txt`. To refresh it with the latest IEEE data:

```sh
make update-oui
make build-all
```

MACs with the **locally administered bit** set (second least-significant bit of the first octet) are shown as `Local/Randomized` regardless of the OUI table — these addresses are self-assigned and have no registered manufacturer.

## License

MIT
