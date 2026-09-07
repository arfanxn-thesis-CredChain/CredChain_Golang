# CredChain — VPS Deployment Runbook

Authoritative, executable deployment guide for AI agents and engineers. Target: a fresh Ubuntu 24.04 VPS. Live deployment: **https://credchain.web.id** (Tencent Lighthouse, `43.133.135.133`).

`CredChain_Golang` is the orchestrator for the whole monorepo. `make prod-up` **pulls** prebuilt images and runs the full stack — it never builds on the VPS.

---

## Architecture

```
Internet
  │
  ▼
nginx :80 (ACME + 301→https) / :443 (TLS)          ← only public container
  ├── /       → react:80        (SPA; calls /api via relative path)
  └── /api/   → golang:8080  ──→ python:8081        (AI extract/embed)
                     │
                     ├── postgres:5432   ┐
                     ├── mongo:27017     ├─ internal only, bound to 127.0.0.1
                     └── anvil:8545      ┘  (dev chain; thesis/demo choice)
```

Same-origin: React talks to `/api` (relative), so a domain change needs **no rebuild**. Blockchain is backend-internal (Go signs txs against anvil) — no MetaMask / public RPC.

### Images

| Image | Source | Built where |
|---|---|---|
| `arfanxn/credchain-golang:latest` | custom | Mac (`make build-push`) |
| `arfanxn/credchain-react:latest` | custom | Mac (`make build-push`) |
| `arfanxn/credchain-python:latest` | custom | Mac (`make build-push`) |
| `nginx:alpine` | official | — |
| `postgres:15-alpine` | official | — |
| `mongo:8.0` | official | — |
| `ghcr.io/foundry-rs/foundry` (anvil) | official | — |

**Never build on the VPS.** Build + push from the Mac; the VPS pulls.

---

## Host prerequisites (VPS)

Installed once in Phase B: `docker` + compose plugin, `node` 20 + `npm` (hardhat contract deploy runs on host), `git`, `make`, `python3`, `certbot`.

---

## Phase A — Code changes (Mac, one-time; already in the repos)

These are committed to `master` and baked into the images. Listed so an agent redeploying from scratch knows what must be present.

**1. `Makefile` — `build-push` target** (build + push all 3 images, no local deploy):
```make
build-push:
	docker buildx build --platform linux/amd64,linux/arm64 -t arfanxn/credchain-golang:latest --push .
	docker buildx build --platform linux/amd64,linux/arm64 \
		--build-arg VITE_GOOGLE_CLIENT_ID="$$(grep -E '^VITE_GOOGLE_CLIENT_ID=' ../CredChain_React/.env.docker | cut -d= -f2-)" \
		--build-arg VITE_APP_ENV=production \
		--build-arg VITE_SUPPORT_EMAIL="$$(grep -E '^VITE_SUPPORT_EMAIL=' ../CredChain_React/.env.docker | cut -d= -f2-)" \
		-t arfanxn/credchain-react:latest --push ../CredChain_React
	docker buildx build --platform linux/amd64,linux/arm64 -t arfanxn/credchain-python:latest --push ../CredChain_Python
	@echo "build-push: all 3 images built (amd64+arm64) and pushed"
```

**2. `docker-compose.yml` — port hardening.** Bind internal services to loopback so Docker's publish does not bypass the firewall. Only nginx stays public.
```yaml
anvil:    ports: ["127.0.0.1:8545:8545"]
golang:   ports: ["127.0.0.1:8080:8080"]
postgres: ports: ["127.0.0.1:5432:5432"]
mongo:    ports: ["127.0.0.1:27017:27017"]
# python already binds 127.0.0.1:8081; nginx stays "80:80" and "443:443"
```
Also add cert mounts to the nginx service:
```yaml
nginx:
  volumes:
    - ./docker/nginx/nginx.conf:/etc/nginx/nginx.conf:ro
    - /etc/letsencrypt:/etc/letsencrypt:ro
    - /var/www/certbot:/var/www/certbot:ro
```

**3. `docker/nginx/nginx.conf` — TLS.** Two server blocks: `:80` serves the ACME challenge and 301-redirects everything else to https; `:443 ssl` terminates TLS and proxies `/api/`→`golang_upstream`, `/`→`react_upstream`. Certs at `/etc/letsencrypt/live/credchain.web.id/{fullchain,privkey}.pem`. Plus, at the `http {}` level:
```nginx
client_max_body_size 12m;   # app caps credential files at 10MB (feature/credential/credential_request.go)
```

**4. `CredChain_React/docker/nginx/default.conf` — `.mjs` MIME.** pdf.js v5 loads its worker as a module worker, which requires a JS MIME type. Serve `.mjs` correctly:
```nginx
location ~* \.mjs$ {
    default_type text/javascript;
    expires 1y;
    add_header Cache-Control "public, max-age=31536000, immutable";
}
```

Build + push after any code change:
```bash
cd CredChain_Golang
docker login          # user: arfanxn (if not already logged in)
make build-push
```

---

## Phase B — DNS + VPS bootstrap

**DNS (Domainesia / MyDomaiNesia):** add two A records, delete any parking record on `@`:

| Type | Host | Value |
|---|---|---|
| A | `@` | `43.133.135.133` |
| A | `www` | `43.133.135.133` |

Verify from anywhere (propagation 5–60 min):
```bash
dig +short credchain.web.id       # → 43.133.135.133
dig +short www.credchain.web.id   # → 43.133.135.133
```

**Bootstrap (on VPS):**
```bash
# Docker + compose plugin
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER      # GOTCHA #3: log out/in after this

# Node 20 LTS (hardhat contract deploy runs on host)
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

sudo apt-get install -y git make python3 certbot
```

**Firewall:**
- Tencent Lighthouse panel ("Manage Firewall Rules"): allow inbound TCP `22`, `80`, `443` only.
- On VPS: `sudo ufw allow 22 && sudo ufw allow 80 && sudo ufw allow 443 && sudo ufw --force enable`

---

## Phase C — Fetch code + secrets

**SSH key → GitHub** (repos are private under the `arfanxn-CredChain` org):
```bash
ssh-keygen -t ed25519 -C "vps-credchain" -f ~/.ssh/id_ed25519 -N ""
cat ~/.ssh/id_ed25519.pub          # add to GitHub → Settings → SSH keys
ssh -T git@github.com              # expect "Hi ... successfully authenticated"
```

**Clone all 4 repos as siblings:**
```bash
mkdir -p ~/credchain && cd ~/credchain
for r in Golang React Python Solidity; do
  git clone git@github.com:arfanxn-CredChain/CredChain_$r.git
done
```

**Copy gitignored secrets from Mac** (clone does NOT include env files). Run these FROM the Mac:
```bash
cd /Users/arfanxn/Developments/credchain
scp CredChain_Golang/.env.docker   ubuntu@43.133.135.133:~/credchain/CredChain_Golang/.env.docker
scp CredChain_React/.env.docker    ubuntu@43.133.135.133:~/credchain/CredChain_React/.env.docker
scp CredChain_Python/.env.docker   ubuntu@43.133.135.133:~/credchain/CredChain_Python/.env.docker
scp CredChain_Solidity/.env        ubuntu@43.133.135.133:~/credchain/CredChain_Solidity/.env   # GOTCHA #1
```

**Edit golang `.env.docker` on the VPS** for production:
```bash
GIN_CORS_ALLOW_ORIGINS=https://credchain.web.id
PYTHON_AI_API_KEY=<must equal CredChain_Python/.env.docker API_KEY>   # GOTCHA #6
```

**Fix host bind-mount ownership** (container runs as UID 1000; see GOTCHA #2):
```bash
cd ~/credchain/CredChain_Golang
mkdir -p logs uploads docker/backups
sudo chown -R 1000:1000 logs uploads docker/backups
```

---

## Phase D — TLS + deploy

**Issue the cert (first-boot ordering).** The nginx `:443` block references cert files that don't exist yet, so nginx won't start clean until the cert is issued. Use webroot with the ACME challenge:
```bash
cd ~/credchain/CredChain_Golang
sudo mkdir -p /var/www/certbot
# bring up just enough to serve the challenge on :80 (react + golang + nginx),
# OR obtain standalone before nginx binds :80:
sudo certbot certonly --standalone \
  -d credchain.web.id -d www.credchain.web.id \
  --email arfan2173@gmail.com --agree-tos --no-eff-email
```
> If nginx is already holding :80, use `--webroot -w /var/www/certbot` instead of `--standalone`.

**Deploy the full stack:**
```bash
make prod-up
```
This: creates the `credchain` network; pulls + starts anvil/postgres/mongo; runs `scripts/setup-contracts.py` (`npm ci` + hardhat deploy to anvil, idempotent — writes contract addresses into `.env.docker`); pulls golang; runs Postgres + Mongo migrations; seeds the super-admin; pulls + starts react/nginx/python; prunes dangling images.

**Google OAuth:** in Google Cloud Console → Credentials → OAuth client → **Authorized JavaScript origins**, add `https://credchain.web.id`. No redirect URI needed (that path is dev-only, used by the `get-google-id-token` CLI).

---

## Phase E — Verify + auto-renew

**Renewal cron** (installed non-interactively to avoid GOTCHA #5). cert was standalone-issued, so stop nginx during renewal (GOTCHA #4):
```bash
( sudo crontab -l 2>/dev/null; \
  echo '0 3 * * * certbot renew --pre-hook "docker stop nginx" --post-hook "docker start nginx" --quiet' ) \
  | sudo crontab -
sudo crontab -l
```

**Test renewal path:**
```bash
sudo docker stop nginx && sudo certbot renew --dry-run; sudo docker start nginx
# expect: "all simulated renewals succeeded"
```

**Verify from the Mac:**
```bash
curl -I http://credchain.web.id      # → 301 (redirect to https)
curl -I https://credchain.web.id     # → 200
nc -zv 43.133.135.133 5432           # → refused/timeout (NOT public)
nc -zv 43.133.135.133 8545           # → refused/timeout (NOT public)
```

---

## Redeploy (after code changes)

```bash
# Mac — rebuild + push changed images
cd CredChain_Golang && make build-push

# VPS — pull code + images, restart stack (volumes/data preserved)
cd ~/credchain/CredChain_Golang && git pull
make prod-up
```
`.env.docker` and `CredChain_Solidity/.env` are gitignored — env changes are scp'd or edited on the VPS directly, never pulled. `make prod-fresh` wipes all volumes + uploads before `prod-up` (destructive; only for a clean reset).

---

## Gotchas (learned in production — read before deploying)

1. **`CredChain_Solidity/.env` is required and not cloned.** Filename is `.env` (not `.env.docker`), gitignored. `scripts/setup-contracts.py` runs hardhat, which needs `INITIAL_SUPER_ADMIN_WALLET_ADDRESS` from it. Symptom: `INITIAL_SUPER_ADMIN_WALLET_ADDRESS environment variable is required`. Fix: scp it (Phase C).

2. **Host bind-mount dirs must be owned by UID 1000.** The golang container runs as `app` (UID 1000). Host `logs/`, `uploads/`, `docker/backups/` are root-owned after creation → `open logs/app.log: permission denied` on migrate. Fix: `sudo chown -R 1000:1000 logs uploads docker/backups`.

3. **`usermod -aG docker` needs a fresh session.** Right after adding yourself to the docker group, the current shell still lacks it → `permission denied ... /var/run/docker.sock`. Fix: reconnect SSH (or `newgrp docker`), verify `docker ps`.

4. **cert is standalone-issued → renewal must free port 80.** nginx holds :80/:443, so `certbot renew` can't bind. The cron uses `--pre-hook "docker stop nginx"` / `--post-hook "docker start nginx"`.

5. **`crontab -e` fails on exotic `TERM`.** With `TERM=xterm-ghostty`, `sensible-editor` exits 1 → `Error opening terminal`. Fix: install the line non-interactively by piping to `sudo crontab -` (Phase E).

6. **Go → Python `401 Unauthorized` → empty embedding.** Python validates the `x-api-key` header against its `API_KEY`; Go sends `PYTHON_AI_API_KEY`. If they differ (e.g. Go's is empty), every `POST /extract` returns 401, the extract job retries ~4× then gives up, and the credential shows extraction failed (`extract_failed_at` set) / `python extract returned empty embedding for ...` (issuance itself still succeeds on-chain). Fix: set `PYTHON_AI_API_KEY` = Python's `API_KEY`, then `docker restart golang` — `docker compose up -d golang` may report "up-to-date" and NOT recreate the container, so it keeps the old env in memory. Go reads `.env.docker` only at process start.

7. **`413 Request Entity Too Large` on credential upload.** nginx default `client_max_body_size` is 1 MB, but the app accepts 10 MB (`feature/credential/credential_request.go`). nginx rejects the upload before it reaches Go. Fix: `client_max_body_size 12m;` at the `http {}` level of `docker/nginx/nginx.conf`. This nginx.conf is bind-mounted, so `docker restart nginx` applies it — no image rebuild.

8. **PDF preview shows "File mungkin rusak".** The React nginx served `.mjs` as `application/octet-stream`; pdf.js v5 loads its worker as a module worker and refuses a non-JS MIME (`Failed to load module script: ... non-JavaScript MIME type`). Image previews use a plain `<img>` (no worker) so they always work — the split symptom is the tell. Fix: serve `.mjs` as `text/javascript` in `CredChain_React/docker/nginx/default.conf`, then rebuild + push the react image and pull on the VPS (config is baked into the image, not bind-mounted). The worker asset is cached `immutable` → hard-reload (Cmd+Shift+R) or DevTools "Disable cache" to clear a stale copy. Verify: `curl -sI https://credchain.web.id/assets/pdf.worker.min-*.mjs` shows `Content-Type: text/javascript`.

---

## Related docs

- [README.md](README.md) — human quick reference
- [AGENTS.md](AGENTS.md) — architecture, patterns, conventions
