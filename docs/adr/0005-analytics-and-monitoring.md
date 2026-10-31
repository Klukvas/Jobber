# ADR 0005: Frontend monitoring and analytics

Status: accepted, 2026-10-31.

## Context

The code for Sentry, GA4 and PostHog was in the frontend, but none of it reached the published
bundle: the build arguments stopped short of the image, the CSP refused the hosts, and nothing
reported a conversion. Turning the flags on as they were would have sent a customer's email and
one-time reset code to third parties, because every page view carried `pathname + search`. The
product holds resumes, contacts and job searches, so what is collected is decided here.

## Decisions

| Question | Decision | Why |
| --- | --- | --- |
| Which trackers | GA4 and PostHog | GA4 answers where visitors come from and can feed Google Ads later. PostHog follows an account through the funnel, and retention over more than GA4's two months. |
| PostHog region | EU, `https://eu.i.posthog.com` | Customers are mostly European. A project's region cannot be changed. |
| What PostHog reads | Only events this app sends by name | Autocapture, session recording, heatmaps, dead clicks, surveys and feature flags are off: the screen holds resumes and contacts. |
| Sentry user | Account id, never the email | The id finds the account. The backend sends no user at all. |
| Sentry without consent | Error reports only | No cookie, and the privacy policy describes them as sent when something breaks. |
| Sentry with consent | Traces at 20% too | A sample of page timings. Sampled per trace, so a withdrawal applies to the next one. |
| Sentry sessions | Off for everybody | The SDK can start them after consent but has no way to stop them on withdrawal. |
| Sentry Replay | Off | It records every session into a buffer for every visitor, in the resume editor too. Turning it on needs consent, `worker-src 'self' blob:` (`'self'` for `/sw.js`) and a new policy version. |
| Source maps | Uploaded, then deleted | A minified stack is close to useless. See "Build". |
| Google Ads tag | Not installed | It needs its own consent, CSP entries and a frame. When campaigns start, link Ads to GA4 and import `purchase`, as FluxRadar does. |

## What is sent

Everything goes through `fe/src/shared/lib/analytics.ts`, whose events are a closed list with fixed
parameters: no call site can pass free text.

| Event | Where | Parameters |
| --- | --- | --- |
| `sign_up` | `RegisterModal`, the server accepted the registration | `method` |
| `email_verified` | `RegisterModal` and `LoginModal`, the code was accepted | none |
| `begin_checkout` | `useCheckout`, a checkout URL exists, before the redirect | `currency`, `value`, `items` |
| `purchase` | `AppLayout`, the backend reports a higher plan | `transaction_id`, `currency`, `value`, `items` |

`value` is the list price, $24. The amount paid can differ after a discount, so revenue is read from
Creem. These numbers size the funnel and only cover visitors who gave consent; the database and
Creem remain the source of truth.

**Purchase.** Returning to `?subscription=success` proves nothing: anybody can open that URL, and the
plan comes from the webhook. The event fires where `AppLayout` already detects an upgrade, which
takes a baseline this app wrote before the redirect, read once and then deleted, and a plan the
backend reports as higher. A reload, a shared link and an abandoned checkout report nothing. A
reload while the webhook is still pending loses the event; undercounting is the safe direction.
`transaction_id` is a UUID written with the baseline, so no backend change was needed. A plan granted
by hand in the admin is not a sale.

**Begin checkout.** gtag sends on its own schedule, so an immediate redirect lost the hit. The
redirect waits for gtag's callback, 400 ms at the latest, and not at all without consent.

**Addresses.** `analyticsUrl.ts` is the single cleaner used by GA4, PostHog and Sentry. Only campaign
tags survive in a query. `/s/:token`, record ids and the public share API are collapsed. Another
site's address is reduced to its origin. A test walks the router so a new route with a parameter
has to be handled.

**Where trackers run.** Only on `cursopus.com`. The same bundle runs in the prerenderer during the
Docker build, and in the backend's headless Chrome on `/print/resume` with a customer's resume on the
page. Neither may report.

**Page crashes.** React Router catches a crash inside a page before any boundary of ours: without an
error element it shows its own developer screen and reports nothing. The root route's `RouteError`
rethrows the error, so `GlobalErrorBoundary` reports it and shows the app's own page.

## Consent

`CONSENT_POLICY_VERSION` became 2. The banner used to say no analytics were in use, so an earlier
acceptance was not informed; those visitors are asked again and refusals stand. Bump the version
again whenever the banner's description of what is collected changes, and keep
`fe/src/pages/Privacy.tsx` in step.

**Fonts.** DM Sans, DM Mono and Syne are bundled from `@fontsource` (`fe/src/app/fonts.ts`), because a
stylesheet from Google Fonts hands every visitor's address to Google before the banner is answered.
The `Caddyfile` still allows `fonts.googleapis.com` and `fonts.gstatic.com`, and nothing on the public
path uses them any more: remove both with the next `Caddyfile` change, which needs a write on the
server. `ResumeBuilderPrint` still loads a resume's fonts from Google, but only in the backend's
headless Chrome, where no visitor is involved.

## Build

All public values are GitHub **Variables**, read by `deploy-dev.yml`, because they end up in the
bundle anyway and a secret holding `true` would mask that word across every log.
`SENTRY_AUTH_TOKEN` is the only secret. It reaches the image build as a BuildKit secret, never as a
build argument: `cache-to mode=max` pushes the builder's layers to the registry. Maps are uploaded and
deleted inside one `RUN`, so no layer holds them. A failed upload warns and the build goes on: an
outage at Sentry must not hold back a release while payments are live.

## CSP

`script-src` gains exactly `https://www.googletagmanager.com`. Google picks its regional collection
hosts itself, so `connect-src` and `img-src` carry its documented masks. PostHog needs one
`connect-src` host because nothing is lazy-loaded; removing `advanced_disable_flags` or
`disable_external_dependency_loading` means adding `eu-assets.i.posthog.com` as well. Sentry is the
exact ingest host from the DSN.

## GA4 property settings the code relies on

If one of these changes, the privacy policy may stop being true.

- Enhanced measurement: off. Its site search would read `?search=`, outbound clicks would send links
  to saved jobs, and file downloads would send file names.
- Data retention: 2 months. Google signals: off. User-provided data: off.
- Ads personalization: disallowed.
- Redact data: email on; URL keys `email`, `code`, `token`, `search`.
- Unwanted referrals: `creem.io`, so a return from checkout does not open a new visit.
- Key events: `sign_up` and `purchase`, marked once the first ones arrive.

## Rollout and rollback

The PR changes the `Caddyfile`, `docker-compose.yml` and the workflow, so the release builds and
deploys both services, and preflight compares both files with the server.

1. Enter the variables and the secret in GitHub. Nothing reads them until the merge.
2. Confirm no CI run is active.
3. On the server, back up both files and write the new ones **in place** (`cat new > file`).
   `Caddyfile` is a bind mount of one file: `mv`, `sed -i` and rsync leave the container on the old
   inode. Reload Caddy.
4. Merge. It is a real release, without `[skip ci]`, about 10 minutes with 1 to 3 minutes of downtime.

The new CSP is a superset of the old one, so the old frontend keeps working under it and the order
above never leaves new code behind an old policy. Between steps 3 and 4 any release from `main` stops
at `caddy_drift` or `base_compose_drift`, which is why nothing else may run then.

**Rollback.** Trackers: set the `VITE_FEATURE_*` variables to `false` and run the workflow with
`force_frontend`; `main` still matches the server, so nothing else moves. Re-running an older commit
does not work on its own: preflight would stop at `caddy_drift` until both files on the server match
that commit too. If the release itself is at fault, the previous images are pinned in
`releases/ci-<sha>/rollback-<schema>.yml` on the server. CSP: write the backup back in place and
reload Caddy; `main` must then match again, or the next release stops at `caddy_drift`.

## Checking it after a deploy

Fresh private window, DevTools open:

1. Before answering the banner: no request to Google or PostHog, no `_ga` or `ph_*` cookie, no CSP
   violation in the console.
2. Accept: the visit appears in GA4 Realtime and in PostHog Activity.
3. Cookie settings in the footer: cookies are gone and no further hits leave.
4. A browser driven by automation shows nothing in PostHog: it drops anything reporting
   `navigator.webdriver`.
