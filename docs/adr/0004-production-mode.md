# Production mode: plaintext Postgres on the Docker bridge, explicit CORS origins

`SERVER_ENV=production` hardens the API, but the live server could not boot with it:
the guards in `be/internal/config/config.go` refused `DB_SSL_MODE=disable` and an
empty `ALLOWED_ORIGINS`. So it has been running as `development`. This records how
those guards are met, what the mode changes for each flow, and how to switch.

## What the mode changes

| | `development` | `production` |
| --- | --- | --- |
| Auth cookies | `SameSite=Lax`, no `Secure` | `SameSite=Strict; Secure` |
| CORS | `*`, no credentials | listed origins, with credentials |
| gin | debug mode | release mode |
| `/swagger/*` | served by the API | not registered (nginx never proxied it anyway) |
| Extra startup guards | none | origins, JWT secrets of 32+ characters, database TLS |
| Sentry `environment` | `development` | `production` |

Nothing else branches on the mode. `IgnoresLiveBilling` only logs a warning.

## Postgres stays without TLS on the Docker bridge

**Decision:** keep `sslmode=disable` for a database reached over loopback or by a
single-label name, and require TLS for every other host.

The guard exists to stop database traffic crossing a network someone else shares.
Here it does not: `backend` and `postgres` run on one host, talk over the
`jobber-network` bridge, and Postgres publishes no port. TLS would cost more than
it buys on that path:

- `require` does not check the server's identity, so it stops passive sniffing and
  nothing else. Sniffing this bridge takes root on the host, or a compromised
  container, and either one can read `.env` or attack Postgres directly.
- `verify-full` would be meaningful, but needs a private CA, a certificate for the
  name `postgres`, rotation, and a Postgres restart. A forgotten expiry becomes an
  outage on a path that never leaves the machine.
- Redis uses the same bridge in plaintext. TLS for Postgres alone would not change
  what a process on that bridge can reach.

The rule is tied to the host rather than to a flag, so it cannot outlive the
setup it was written for. A move to a managed database or a second machine means a
dotted name or an IP, and the API refuses to boot until TLS is on. Private IPs do
not count as internal, because they can sit on a network spanning machines.

The guard also got tighter: `prefer` and `allow` used to pass it, and against a
server with TLS off they connect in plaintext without a word. Outside the internal
case only `require`, `verify-ca` and `verify-full` are accepted.

## `ALLOWED_ORIGINS`

```
https://cursopus.com,chrome-extension://koegfmmcpedfgnjnohcaieecdoflmlab
```

| Origin | Listed | Why |
| --- | --- | --- |
| `https://cursopus.com` | yes, first | The SPA. The first entry is also the fallback for `PUBLIC_BASE_URL`; with the extension first the API refuses to boot. |
| `chrome-extension://koegfmmcpedfgnjnohcaieecdoflmlab` | yes | The Web Store extension, see below. |
| `https://www.cursopus.com` | no | Caddy redirects it to the apex, so no page runs there. |
| `https://jobber-app.com` | no | Pages redirect to the apex. `/api/*` stays for old extension builds, which send the extension origin. |
| `http://localhost:*` | no | Development only. |

The extension sends a bearer token and no cookies, from its service worker and side
panel; content scripts do not call the API. Its `host_permissions` exempt it from
CORS, so a normal install works with or without the entry. The entry is for a user
who restricts its site access in Chrome: the exemption goes away, and with an
explicit list `fetch` fails where `*` used to let it through. The origin can only be
produced by that extension, and it uses no cookies, so listing it opens nothing.

The API never rejects a request for its `Origin`: CORS only decides what a browser
lets a page read. Admin writes are the exception and already require a trusted origin.

## Cookies under `SameSite=Strict`

`Strict` withholds cookies from the document request of a navigation that starts on
another site. The SPA does not need them there: the document is static, and the
`/api/v1/session` call that follows comes from a `cursopus.com` page, so it is
same-site and carries the cookie. `Strict` therefore stays.

| Flow | Result |
| --- | --- |
| Return from Creem to `/app/settings?subscription=success` | Stays signed in. |
| The same return with an expired access token | `/auth/refresh` gets its cookie, session recovers. |
| Google Calendar callback | Uses OAuth `state`, needs no cookie; the app behind the redirect stays signed in. |
| Password reset link in an email | A public page, needs no session. It is the only link in our emails. |
| Google sign-in | Does not exist; sign-in is email and password. |
| Extension opening a job in a tab | Same as any outside link. |
| Session issued before the switch | Keeps working; refresh replaces its cookies without duplicates. |
| PDF rendering | The print page gets its data injected and makes no API call. |

`Secure` is safe behind the proxies: the browser only ever sees HTTPS, the API sets
the attribute regardless of the proxied scheme.

## Checked locally (2026-10-30)

The API ran with `SERVER_ENV=production` and `DB_SSL_MODE=disable` on a Docker network
with the service names `postgres` and `redis`, migrated to v54, against the built SPA.
A real Chrome drove every flow above from a page on another site, and the extension
was loaded unpacked:

- without its origin listed and with site access withheld, the extension got
  `Failed to fetch`; with its origin listed, login, session, jobs and refresh passed;
- with host permission it passed either way;
- `/swagger/index.html` answered 404, the log had no `GIN-debug` line, a foreign site
  could not read the API;
- `*`, an empty list, a dotted `DB_HOST` without TLS, `prefer`, a private IP and a
  short JWT secret each refused to boot.

Not exercised: real Creem, Google and mail hosts (an outside page stood in for them),
Safari and Firefox, real HTTPS (Chrome treats `localhost` as secure), and the extension
under its store ID. The smoke test below covers these.

## Switching the server

Do this after the Creem go-live has settled. Every step marked **[go-ahead]** changes
the server and needs its own explicit approval.

1. **Ship the code.** Merge this change without `[skip ci]`: `be/**` triggers a backend
   release. With `SERVER_ENV=development` the new guard is inert, so behaviour does not
   change. `docker-compose.yml` and `Caddyfile` are untouched, so `release.sh` sees no drift.
2. **Read-only check.**
   - the running backend image is the release from step 1;
   - both JWT secrets are 32+ characters. Lengths only:
     `docker exec jobber-backend sh -c 'echo ${#JWT_ACCESS_SECRET} ${#JWT_REFRESH_SECRET}'`.
     If either is shorter, stop: replacing a secret signs everyone out, and that is a
     separate decision;
   - `DB_HOST=postgres` and `DB_SSL_MODE` is `disable` in the container;
   - `docker port jobber-postgres` and `docker port jobber-redis` print nothing;
   - `LOG_LEVEL` is known. The Compose default is `debug`;
   - whether Sentry is on, and whether any alert filters on `environment: development`.
3. **Back up [go-ahead].** Copy `.env` and `docker-compose.deploy.yml` to
   `backups/pre-production-mode-<stamp>/`, mode 600.
4. **Edit `.env` [go-ahead].** Set the three keys below, leave `DB_SSL_MODE` alone, and
   check the render with `docker compose -f docker-compose.yml -f docker-compose.deploy.yml config`
   before moving on. A running container ignores `.env` until it is recreated.

   ```
   SERVER_ENV=production
   ALLOWED_ORIGINS=https://cursopus.com,chrome-extension://koegfmmcpedfgnjnohcaieecdoflmlab
   LOG_LEVEL=info
   ```
5. **Recreate the backend [go-ahead].** About 10-20 seconds of API downtime. The nginx
   reload is required: the recreated container has a new IP, and nginx holds the old one.

   ```bash
   docker compose -f docker-compose.yml -f docker-compose.deploy.yml up -d --no-deps backend
   docker exec jobber-frontend nginx -s reload
   ```
6. **Postflight, read-only.**
   - health is green, the startup log says `"env":"production"`, there is no `GIN-debug`
     line and no error;
   - an unauthenticated API call answers 401;
   - `POST /api/v1/webhooks/creem` with a bad signature answers 400;
   - `curl -s -D - -o /dev/null -H 'Origin: <origin>' https://cursopus.com/api/v1/session`
     returns that origin in `Access-Control-Allow-Origin` for both listed origins, and
     nothing for `https://evil.example`.
7. **Browser smoke test, by hand.**
   - a browser signed in before the switch is still signed in;
   - after a fresh sign-in, DevTools shows both cookies as `Secure` and `Strict`;
   - Settings, then the Creem billing portal, then back: still signed in;
   - a password reset link opened from a real inbox;
   - the store extension: open the panel, list jobs, save a job;
   - sign-in and an outside link in Safari.

## Rollback

Restore `.env` from step 3, then run the two commands from step 5. It takes about a
minute and needs no image change, since the new guard is inert in `development`.

Sessions survive in both directions: JWT secrets do not change and no schema changes,
and cookies are replaced at the next refresh. If the backend fails to boot after step 5,
`docker logs --tail 20 jobber-backend` names the guard in `Failed to load configuration`;
roll back first and read the reason second.

## Leftover

`docker-compose.yml` still says `must be 'require' in production` next to `DB_SSL_MODE`.
The comment is out of date, but `release.sh` compares the server's Compose file with
`main` by hash (`base_compose_drift`), so correcting it here would block the release
until the file is copied to the server by hand. Fix it with the next Compose change.
