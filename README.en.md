# MTPanel (EN)

Self-hosted control panel for **TeleMT** on Linux.

## Contents

1. [Overview](#overview)
2. [Features](#features)
3. [Requirements](#requirements)
4. [One-line Install](#one-line-install)
5. [Installer Flags](#installer-flags)
6. [What Is Automated](#what-is-automated)
7. [First Run](#first-run)
8. [Screenshots](#screenshots)
9. [Full Uninstall](#full-uninstall)
10. [Diagnostics](#diagnostics)

## Overview

MTPanel provides a simple flow:

- install with one command;
- open web UI;
- install/start/restart TeleMT;
- generate and revoke proxy links.

## Features

- First-run setup and authentication.
- TeleMT lifecycle management.
- One-click `tg://proxy?...` link generation.
- Logs and health checks.
- TeleMT update check and apply flow.
- Automatic firewall setup in installer.

## Requirements

- Linux with `systemd`.
- Root/sudo access.

## One-line Install

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash
```

With explicit ports:

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash -s -- --port 8080 --proxy-port 443
```

## Installer Flags

- `--port <port>`: panel port (default `8080`)
- `--proxy-port <port>`: TeleMT port (default `443`)
- `--panel-allow <CIDR>`: panel access CIDR (example: `1.2.3.4/32`)
- `--repo <owner/repo>`: release source repo

## What Is Automated

Installer will:

1. detect OS/arch/deps;
2. download/build backend and frontend;
3. write `/etc/mtpanel/config.json`;
4. remove stale `mtpanel.service.d` overrides;
5. install a stable `mtpanel.service`;
6. auto-apply firewall rules:
   - proxy port open to all;
   - panel port restricted to SSH source IP if detected.

## First Run

1. Open `http://<SERVER_IP>:8080`.
2. Go to `/setup` and set admin password.
3. Login via `/login`.
4. Install TeleMT in `Proxy` section.

## Screenshots

Put screenshots into `docs/screenshots/`:

- `dashboard.png`
- `proxy.png`
- `links.png`
- `updates.png`

## Full Uninstall

```bash
sudo systemctl stop mtpanel 2>/dev/null || true
sudo systemctl disable mtpanel 2>/dev/null || true
sudo pkill -f '/opt/mtpanel/mtpanel' || true

sudo systemctl stop telemt 2>/dev/null || true
sudo systemctl disable telemt 2>/dev/null || true

sudo rm -f /etc/systemd/system/mtpanel.service
sudo rm -rf /etc/systemd/system/mtpanel.service.d
sudo rm -f /etc/systemd/system/telemt.service
sudo systemctl daemon-reload

sudo rm -rf /opt/mtpanel /opt/telemt
sudo rm -rf /etc/mtpanel /etc/telemt
sudo rm -rf /var/lib/mtpanel

sudo userdel mtpanel 2>/dev/null || true
```

## Diagnostics

```bash
sudo systemctl status mtpanel --no-pager -l
sudo journalctl -u mtpanel -n 120 --no-pager -l
curl -I http://127.0.0.1:8080/
```
