<p align="center">
  <img src="assets/banner.png" alt="arpdvark" width="400">
</p>

# arpdvark

[![CI](https://github.com/fabioconcina/arpdvark/actions/workflows/ci.yml/badge.svg)](https://github.com/fabioconcina/arpdvark/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/fabioconcina/arpdvark)](https://goreportcard.com/report/github.com/fabioconcina/arpdvark)
[![GitHub release](https://img.shields.io/github/v/release/fabioconcina/arpdvark)](https://github.com/fabioconcina/arpdvark/releases/latest)
[![Go version](https://img.shields.io/github/go-mod/go-version/fabioconcina/arpdvark)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> **Experimental project** — built to learn Go, TUI development, and raw networking. Intended for use on home networks only. Not hardened for production or adversarial environments.

A minimal, fast terminal-based network inventory tool. Scans your local network using ARP, identifies connected devices, resolves hostnames and MAC vendors, and displays results in a full-screen auto-refreshing TUI.

```
arpdvark  •  interface: eth0  •  subnet: 192.168.1.0/24

IP Address       MAC Address         Hostname        Label     Vendor
192.168.1.1      30:1f:48:10:f3:04   router.home     router    zte corp.
192.168.1.2      a8:1d:16:31:a6:4f   laptop.home               AzureWave
192.168.1.5      98:e2:55:7f:8a:48                   switch    Nintendo
192.168.1.112    34:af:b3:82:16:95   echo.home       echo      Amazon Tech.
192.168.1.136    ea:03:65:53:c9:62                              Local/Random.

5 device(s)  •  last scan: 2s ago  •  e: label  r: rescan  q: quit
```

**Supported platforms:** Linux (amd64, arm64)

## Features

- **ARP scanning** — discovers all active hosts on the local subnet using raw ARP packets
- **Hostname resolution** — tries three methods in order: system DNS, gateway DNS (dnsmasq/home routers), mDNS unicast (Bonjour/Avahi)
- **MAC vendor lookup** — identifies hardware vendors from the embedded [IEEE OUI registry](https://standards-oui.ieee.org/oui/oui.txt) (~39k entries)
- **Locally administered MAC detection** — flags randomized, VM-assigned, or manually set MACs as `Local/Randomized` (these have no OUI entry by design)
- **Full-screen TUI** — fills the terminal, resizes dynamically, auto-refreshes on a configurable interval
- **Persistent device table** — devices seen in previous scans remain visible; `LastSeen`, MAC, and vendor are updated on each round
- **Device persistence** — scan results are saved to `~/.config/arpdvark/state.json` across runs, tracking first-seen/last-seen timestamps and online/offline status; the TUI shows previously known devices immediately on startup
- **Host labels** — assign custom names to any device; labels are keyed by MAC address and persist across restarts in `~/.config/arpdvark/tags.json`
- **Multi-round first scan** — the first ARP sweep sends 3 rounds of requests (100 ms apart) to catch slow responders such as Wi-Fi clients in power-save mode; subsequent scans send a single round since the device table accumulates across sweeps
- **Column sorting** — use `←`/`→` to cycle sort column (IP, MAC, Hostname, Label, Vendor, Last Seen); press `s` to toggle ascending/descending
- **Device filtering** — press `/` to filter the device table by any field; matches IP, MAC, hostname, label, and vendor; press `/` again to clear
- **New device alerts** — devices seen for the first time (not in the state file from previous runs) are highlighted in green in the TUI, with a count in the status bar
- **Activity heatmap** — the detail view shows a weekly activity pattern (7 days x 24 hours) built from scan history, using block characters to visualize when a device is typically connected; activity is recorded in all modes (TUI, `--json`, `--once`) and persisted to `~/.config/arpdvark/activity.json`
- **Rate-limited scanning** — ARP requests are rate-limited (1000 pkt/s for /24 and smaller, 5000 pkt/s for larger subnets) to avoid overwhelming switches or triggering IDS alerts

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
       arpdvark forget [--older-than N] [MAC ...]

Options:
  -i <interface>    Network interface to scan (default: auto-detect)
  -t <seconds>      Refresh interval in seconds (default: 10)
  --large           Allow scanning subnets larger than /16
  --json            Run one scan and output JSON to stdout
  --once            Run one scan and print a plain-text table to stdout
  --all             Include offline devices (--json and --once only)
  --mcp             Run as MCP server (stdio transport)
  --version         Print version and exit
  -h                Show help
```

### Interactive TUI (default)

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
| `e` | Edit label for selected row |
| `←` / `→` | Cycle sort column (IP, MAC, Hostname, Label, Vendor, Last Seen) |
| `s` | Toggle sort direction (ascending / descending) |
| `/` | Open filter (Enter to apply, Esc to clear, `/` again to clear) |
| `o` | Toggle show/hide offline devices |
| `Enter` | Open device detail view |
| `Esc` / `Enter` | Close detail view / close filter / cancel label edit |

**Detail view** (`Enter` on a row): shows all device fields untruncated — IP, MAC, hostname, label, vendor, status, first seen, last seen — plus a weekly activity heatmap showing when the device is typically online. Navigate fields with `↑`/`↓`.

### JSON output (`--json`)

```sh
sudo arpdvark --json
sudo arpdvark --json -i eth0
```

Runs a single scan, prints a JSON array to stdout, and exits. Suitable for piping to `jq`, scripts, or other tools.

```json
[
  {
    "ip": "192.168.1.1",
    "mac": "aa:bb:cc:dd:ee:ff",
    "vendor": "Cisco Systems",
    "hostname": "router.local",
    "label": "main-router",
    "first_seen": "2024-01-01T00:00:00Z",
    "last_seen": "2024-01-01T00:00:00Z"
  }
]
```

Use `--all` to include offline (previously seen) devices:

```sh
sudo arpdvark --json --all
```

With `--all`, each device includes an `"online"` field (`true`/`false`).

### Plain-text table (`--once`)

```sh
sudo arpdvark --once
```

Runs a single scan, prints a tab-aligned text table to stdout, and exits. Parseable with `awk`/`cut`.

### MCP server (`--mcp`)

```sh
arpdvark --mcp
```

Runs an [MCP](https://modelcontextprotocol.io) (Model Context Protocol) server on stdio, exposing a `scan_network` tool. This allows AI agents (e.g. Claude Desktop) to scan your network programmatically.

**Claude Desktop configuration:**

```json
{
  "mcpServers": {
    "arpdvark": {
      "command": "sudo",
      "args": ["/path/to/arpdvark", "--mcp"]
    }
  }
}
```

### Forget devices

Remove specific devices from the state file, or prune devices not seen in a given number of days:

```sh
arpdvark forget aa:bb:cc:dd:ee:ff           # remove one device by MAC
arpdvark forget --older-than 30             # remove devices unseen for 30+ days
```

### Activity tracking

Every scan (TUI, `--json`, `--once`) records which devices are online, building a weekly activity heatmap visible in the detail view (`Enter` on a device). To get useful data, run periodic scans in the background with a cron job:

```sh
# Add to root's crontab (sudo crontab -e):
*/5 * * * * SUDO_USER=youruser /path/to/arpdvark --once > /dev/null 2>&1
```

The `SUDO_USER` variable ensures activity and state data are stored in your home directory (`~youruser/.config/arpdvark/`) rather than root's.

Activity data is stored in `~/.config/arpdvark/activity.json`. The `forget` subcommand also removes activity data for forgotten devices.

### Exit codes

Exit codes apply to `--json` and `--once` modes:

| Code | Meaning |
|------|---------|
| 0 | Success (devices found) |
| 1 | Error (permissions, interface not found, scan failure) |
| 2 | Scan completed but no devices found |

## Permissions

arpdvark uses raw ARP sockets (`AF_PACKET`) and requires elevated privileges. Run with `sudo`, or grant `CAP_NET_RAW` to avoid it:

```sh
sudo setcap cap_net_raw+ep ./arpdvark
./arpdvark
```

## Hostname resolution

Hostnames are resolved concurrently for all discovered devices after each ARP sweep. Three methods are tried in order:

1. **System resolver** — uses `/etc/resolv.conf`. Works if your DNS server (e.g. Pi-hole with DHCP enabled) maintains PTR records.
2. **Gateway DNS** — queries the default gateway (read from `/proc/net/route`, falling back to the first host in the subnet) directly on port 53. Home routers running dnsmasq create PTR records for DHCP leases automatically.
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
