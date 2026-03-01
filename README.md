# arpdvark

A minimal, fast terminal-based network inventory tool. Scans your local network using ARP, identifies connected devices, resolves MAC vendors, and displays results in a clean auto-refreshing TUI.

```
╭──────────────────────────────────────────────────────────────────╮
│ arpdvark  •  interface: eth0  •  subnet: 192.168.1.0/24          │
│ IP Address        MAC Address           Vendor                    │
│ 192.168.1.1       a4:c3:f0:12:34:56     Apple, Inc.              │
│ 192.168.1.42      b8:27:eb:aa:bb:cc     Raspberry Pi Foundation  │
│ 192.168.1.101     dc:a6:32:11:22:33     Unknown                  │
│ 3 device(s)  •  last scan: 2s ago  •  r: rescan  q: quit         │
╰──────────────────────────────────────────────────────────────────╯
```

**Supported platforms:** Linux (amd64, arm64)

## Installation

**Pre-built binary** (Linux amd64/arm64):

Download the latest release from the [releases page](https://github.com/yourname/arpdvark/releases) and place the binary in your `PATH`.

**From source:**

```sh
git clone https://github.com/yourname/arpdvark
cd arpdvark
make build
```

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
sudo arpdvark -i eth0 --large
```

**Key bindings:**

| Key | Action |
|-----|--------|
| `q` / `ctrl+c` | Quit |
| `r` | Force immediate rescan |
| `↑` / `↓` | Navigate table rows |

## Permissions

arpdvark uses raw ARP sockets (`AF_PACKET`) and requires elevated privileges. Run with `sudo`, or grant `CAP_NET_RAW`:

```sh
sudo setcap cap_net_raw+ep ./arpdvark
```

## Updating the OUI database

The bundled vendor database is a trimmed snapshot of the [IEEE OUI registry](https://standards-oui.ieee.org/oui/oui.txt). To refresh it:

```sh
make update-oui
make build
```

## License

MIT
