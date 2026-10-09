# Billing moves to Creem, with purchases linked to users by checkout metadata

Supersedes [ADR-0002](0002-fastspring-billing.md). Jobber's billing provider is
**Creem** (Merchant of Record). FastSpring is removed entirely: there were no
paying FastSpring subscribers at cut-over, so no customer or payment data was
migrated and no dual-provider layer exists. Migration `000046` resets any
test-mode row that still carries a FastSpring identifier.

## What changed in shape

| | FastSpring (ADR-0002) | Creem |
| --- | --- | --- |
| Checkout | Store Builder Library popup opened with a session id | Full-page redirect to a Creem-hosted checkout URL |
| Return | popup close callback + `?subscription=success` | `success_url` → `/settings?subscription=success` |
| Webhook | batch of events, base64 HMAC in `X-FS-Signature`, 200/202/503 ack | one event per POST, hex HMAC-SHA256 in `creem-signature`, 200 = delivered |
| Retries | 7 days, 12 attempts | 5 attempts, **24 hours** |
| Missed events | pull `GET /events/unprocessed` (reconciliation sweep) | none available — see below |
| Buyer ↔ user | order tag + HMAC proof, account lookup key | checkout `metadata` (server-written) |

The consequence is a lot less code: the order-tag proof, the account read-back,
the popup storefront derivation, the CSP allowances for a third-party script and
the reconciliation sweep are all gone.

## The purchase → user chain

1. `POST /api/v1/subscription/checkout-session {plan}` — authenticated. The
   backend ensures the user's `subscriptions` row exists, then calls
   [`POST /v1/checkouts`](https://docs.creem.io/api-reference/endpoint/create-checkout)
   with `product_id`, `success_url`, `customer.email`/`name` from the
   **authenticated user record**, and `metadata.jobber_user_id` = the user UUID.
   The browser chooses a plan name and nothing else.
2. The backend returns `{checkout_url}`; the frontend validates it as an absolute
   `https:` URL (the backend does too) and navigates there.
3. Creem sends `checkout.completed` and `subscription.*` events carrying the
   metadata. The owner is resolved by, in order: the provider subscription ID
   already on a row, `metadata.jobber_user_id` (must parse as a UUID, otherwise it
   is treated as absent and never reaches a query), then the Creem customer ID
   already on a row. The product is checked **before the customer hop only**:
   the metadata is written by our server, so it stands for a purchase of ours
   even if product IDs were reconfigured since, while the customer identifies a
   *person* — another product sold to a customer already linked here must not be
   applied to their Jobber row.
4. `checkout.completed` links the customer ID (`subscriptions.external_account_id`)
   to the user and grants nothing. Entitlement comes from `subscription.*` alone.

### Why metadata needs no proof

ADR-0002 needed an HMAC proof because FastSpring's storefront library lets any
visitor attach tags to their own order, so a tag inside a signed webhook only
proved *FastSpring sent it*, never *who wrote it*. Creem has no equivalent: the
checkout is created server-side with our API key, its metadata is fixed at
creation, so a buyer cannot add or change it through our API (Creem's docs do
not say whether the hosted page lets a buyer edit the pre-filled email; see the
checklist). A buyer cannot make a checkout that claims another user. A purchase made without our API
(a Creem storefront or payment link) carries no `jobber_user_id` and therefore
claims nobody.

> ⚠️ Assumption to verify (checklist below): that checkout `metadata` is copied
> onto the **subscription** object, so `subscription.*` events carry it. Creem's
> webhook samples show a `metadata` object on subscriptions, but the docs do not
> state the inheritance. The design degrades safely if it is false —
> `checkout.completed` still links the customer, and every later event resolves
> through it — but the very first `subscription.paid` could arrive before that
> link exists. It is then **retried** (see "unresolvable" below) and lands once
> the link is written.

## Event handling

Actionable: `subscription.active`, `.paid`, `.update`, `.trialing`, `.paused`,
`.scheduled_cancel`, `.past_due`, `.unpaid`, `.canceled`, and `checkout.completed`
(link only). The new state is derived from the object's `status`, not from the
event type:

| Creem `status` | Internal | Notes |
| --- | --- | --- |
| `active`, `trialing` | `active` | |
| `scheduled_cancel` | `active` + `cancel_at = current_period_end` | cancelled but paid through the period |
| `past_due` | `past_due` | dunning; keeps the grace window |
| `paused`, `unpaid` | `paused` | plan stays on the row (recovery restores it), paid quotas stop |
| `canceled` | `cancelled`, plan `free`, no period | consults no product, so a catalog mistake cannot keep access alive |
| anything else | event **fails** (retried) | never guessed |

A field the payload omits keeps its stored value: some Creem samples carry no
period dates, and writing them as `NULL` would erase the renewal date. A
`scheduled_cancel` with no period end falls back to the stored one so the UI
never loses the cancellation date. An ended subscription has no current period.

Deliberately **not** acted on:

- `subscription.expired` — Creem keeps `status: active` while it retries the
  charge; the subscription is over only when `subscription.canceled` follows.
- `refund.created` / `dispute.created` — a refund is not a cancellation (a partial
  refund leaves the subscription billing). They are logged at `warn` so money
  leaving the account is never invisible; ending the subscription is a merchant
  decision made in the dashboard and arrives here as `subscription.canceled`.
- everything else (`credits.*`, …) — acknowledged, `info` log.

### What stays from ADR-0002

These guards were never about FastSpring and are unchanged:

- **Signature first**, against the raw body, before parsing or any query; compared
  as decoded bytes in constant time. The route is public and rate limited, and the
  limiter fails open.
- **One atomic write per event**: the `webhook_events` claim and the
  `subscriptions` write are a single statement, so a failed write never leaves an
  event marked processed.
- **Replays cannot resurrect**: ordering uses the object's own `updated_at`
  (falling back to the envelope's `created_at`), strictly greater-than, enforced
  in the `UPDATE ... WHERE` and not in a prior read. A manual resend gets a fresh
  event ID and a fresh envelope timestamp but not a fresh `updated_at`.
  **One exception: a cancellation that ties** with the applied state still lands
  (and is not re-applied over an existing cancellation). Creem is not documented
  to bump `updated_at` between the last payment and the cancel, and dropping it
  would keep a non-paying user on a paid plan. The envelope fallback and the
  object's `updated_at` come from different clocks, so a row that has seen both
  can misorder; the checklist confirms `updated_at` is always present.
- **One subscriber, one subscription**: a second checkout is refused with
  `409 ALREADY_SUBSCRIBED`, and the atomic write refuses to replace a still-billing
  `external_subscription_id` (`ErrSubscriptionLinkConflict`). What happens next
  depends on whether a redelivery could ever do better:
  - a **non-cancellation newer than the row is retried (HTTP 500)**: the usual
    cause is a delayed cancellation of the old subscription, and once it lands the
    same event applies by itself. Acknowledging would lose a paying user's new
    plan. If the old subscription really is alive the retries run out after 24
    hours, each attempt logged at `error` — the signal that somebody may be paying
    twice;
  - a **cancellation** is acknowledged (`errNothingToEnd`, `info`): it is the replay
    of an old subscription's cancel, or the cancel of a second subscription that was
    never linked, and there is nothing in Jobber to end;
  - a paid event **not newer than the row** is acknowledged and flagged at `warn`
    (`errSecondSubscription`): the older state can never win, and it is what a user
    paying for two subscriptions looks like.

  The same flag is raised when a replacement subscription loses the ordering guard
  to the *previous* subscription's later cancellation (S2's events carry
  `updated_at` T2, S1 was cancelled at T3 > T2). Ordering across two subscriptions
  is meaningless, but relaxing it would let a replay of an old subscription's
  event resurrect it, which is worse; the flag makes the loss visible instead.
  This only arises when a user buys a second subscription outside the app while
  one is live (the app refuses with 409). The UI routes a subscriber to
  change-plan.
- **Environment guard**: an object whose `mode` disagrees with `CREEM_ENVIRONMENT`
  (`prod` = live; `test` and `sandbox` = test; the webhook reference's samples also
  show `local`, treated as test) is acknowledged and never applied, logged at
  `error`. Because that loses the event, startup logs a loud warning when billing
  is on in production with `CREEM_ENVIRONMENT=test` (a hard failure would block
  exercising test mode on the production host before go-live). An object with no `mode` at all cannot be placed and is let
  through rather than guessed at.
- **Unknown product never pays**: an `active` event for a product that is neither
  configured plan fails and is retried.
- `external_account_id` keeps its partial UNIQUE index (one Creem customer ↔ one
  user); a collision is `ErrBillingAccountTaken`, acknowledged and logged at
  `error` (that purchase grants nothing until a person decides).
- **A user with no `subscriptions` row** is created one (`EnsureFree`) when a
  checkout's metadata names them, so a customer who paid is not retried into the
  ground over a missing row. A user who **no longer exists** (FK violation) is
  acknowledged and logged at `error` (`model.ErrUserNotFound`): no retry brings an
  account back.
- **A payload that omits its period** keeps the stored one only while the stored
  end is still ahead of the event. A stale stored end (renewals that carried no
  dates never advanced it) is dropped rather than turned into a cancellation date
  in the past. A customer the event does not carry is left untouched by the write.

### "Unresolvable" is two different things

The metadata is checked first (it is written by our server, so it stands for a
purchase of ours even if product IDs were reconfigured since: such an event fails
and is retried, never dropped). Only **after** it does the product matter: an
event with no Jobber metadata is **foreign** when its product is not Jobber's,
*including* when its customer is already linked here — the customer identifies a
person, not a purchase, and another product they bought must never be applied to
their Jobber row. Such an event is acknowledged and dropped. Otherwise it is
**foreign** when its product is not Jobber's (the Creem account may sell other
things) and is acknowledged and dropped. When the product *is* ours it is
**retryable**: a payload for our own product that we
failed to place is our bug, and `checkout.completed` may still be on its way to
supply the link.

## Acknowledgement

Creem treats `200` as delivered and retries anything else. So: applied, duplicate,
superseded and deliberately skipped events answer `200`; a failed event answers
`500`; a bad signature or unparseable body answers `400`; a body over 1 MiB answers
`413` before it is hashed; a missing webhook secret answers `503` (our
misconfiguration must not look like a client error — in production it is
unreachable, since startup refuses to boot without the secret when ingestion is
on). With ingestion switched off the route is not registered at all and answers
`404`, which Creem retries as a failure for 24 hours.

Dropped-for-good events are logged at **`error`**, because Creem never redelivers
them and the log is the only trace that a paying user may be on the wrong plan:
an environment mismatch, and a Creem customer already linked to another user
(`ErrBillingAccountTaken`, which means that purchase grants nothing until a person
decides), and a user who no longer exists. A refund or dispute and a paid event
for a second subscription log at `warn`; routine skips at `info`.

## There is no reconciliation sweep, on purpose

FastSpring let us pull events it still had no acknowledgement for; ADR-0002's
6-hourly sweep used that. Creem exposes no event listing, and its retries stop
after 24 hours. A deployment that is down, unreachable or holding a stale secret
for longer than that **keeps subscription rows nothing will correct**.

Mitigations in place: webhook failures are logged at `error`; the webhook route
is rate limited fail-open so a throttle never turns into a lost event (the Redis
call is bounded to 300 ms, so an unreachable Redis does not hold deliveries); and a
**scheduled cancellation whose end date passed more than 48 hours ago stops
counting as paid** (`effectivePlan`), so a lost `subscription.canceled` cannot
keep a paid plan forever. Two days outlasts any late or retried delivery. Other
lost transitions (a missed upgrade, a missed `past_due`) are not covered.

Recovery today is manual: **Creem dashboard → Developers → Webhooks → resend**
for the affected events (the ordering guard makes a resend safe).

If this proves insufficient, the replacement is a state sweep rather than an event
sweep. Creem has both a get-by-ID and a list-subscriptions endpoint, so it is
cheaper than an event sweep would have been: periodically fetch every row with
an `external_subscription_id` that is not `cancelled`, and apply the returned
object through the same pipeline. That is not built, because it would be a second
mechanism to maintain before there is a single paying user.

## Plan changes, cancellation, portal

- Change plan: [`POST /v1/subscriptions/{id}/upgrade`](https://docs.creem.io/api-reference/endpoint/upgrade-subscription)
  with `update_behavior: proration-charge-immediately`. Used for moving down as
  well as up. Creem's managing-subscriptions page says a plan change refunds the
  unused time and tax to the original payment method, so **a downgrade can produce
  a `refund.created`**, which logs the refund `warn` described above — expect one
  per downgrade. The local plan moves only when the resulting `subscription.update`
  arrives.
- Cancel: [`POST /v1/subscriptions/{id}/cancel`](https://docs.creem.io/api-reference/endpoint/cancel-subscription)
  with `mode: scheduled, onExecute: cancel` — end of period. Immediate
  cancellation is not exposed.
- Portal: [`POST /v1/customers/billing`](https://docs.creem.io/features/customer-portal)
  with the stored customer ID. The link is validated as absolute HTTPS and the
  host is not pinned (no documented host contract); the browser navigates the
  current tab.
- The API client has a single circuit breaker (ADR-0002 needed two only because
  the FastSpring store was shared with other products' webhooks).

## Configuration

| Variable | Notes |
| --- | --- |
| `CREEM_API_KEY` | required when `FEATURE_PAYMENTS_ENABLED=true`; test and live keys differ |
| `CREEM_WEBHOOK_SECRET` | required when webhook ingestion is on; the server refuses to start without it |
| `CREEM_ENVIRONMENT` | `test` (default) → `test-api.creem.io`, `live` → `api.creem.io` |
| `CREEM_PRO_PRODUCT_ID`, `CREEM_ENTERPRISE_PRODUCT_ID` | no defaults; an empty one means that plan is not purchasable. At least one is required when payments **or** webhook ingestion is on, or the server refuses to start |
| `TRUSTED_PROXIES` | optional comma-separated IPs/CIDRs whose `X-Forwarded-For` is honoured; empty keeps gin's trust-all default |

`FEATURE_PAYMENTS_ENABLED` (new purchases and plan management) and
`FEATURE_BILLING_WEBHOOK_ENABLED` (ingestion, defaults to the payments flag) stay
independent kill switches, as in ADR-0002. The webhook route is
`POST /api/v1/webhooks/creem`. `success_url` is built from `PUBLIC_BASE_URL`.

## Frontend

`GET /subscription/checkout-config` → `{provider, environment, plans}` (no
storefront). `POST /subscription/checkout-session` → `{checkout_url}`. The hook
records the pre-checkout plan as a baseline, navigates, and on return the layout
polls until the backend reports a strictly higher plan — the redirect back is a
cue to start watching, never proof of payment; only a webhook moves the plan.
A bfcache restore (Back from the checkout) resets the busy state. A Back that
reloads the page instead shows the "activating" overlay until it is dismissed or
times out.

The CSP no longer needs any checkout allowance: navigating to Creem is a top-level
navigation, not a script, frame, request or form post.

## Must be confirmed with one test-mode purchase before go-live

- [ ] **Metadata reaches the subscription events.** Complete a purchase through the
  app's own checkout and confirm `subscription.paid` carries
  `metadata.jobber_user_id`, and that the buyer lands on the paid plan.
- [ ] **`checkout.completed` links the customer** (`external_account_id` is set), and
  the portal opens.
- [ ] **`updated_at` is present and monotonic** on subscription objects, and its
  encoding (RFC 3339 string or epoch milliseconds — both are accepted).
- [ ] **Cancel at period end** produces `subscription.scheduled_cancel` and the app
  shows the end date; the period end then produces `subscription.canceled`.
- [ ] **Plan change** via `/upgrade` produces a `subscription.update` whose product is
  the new one (the parser reads `product`, falling back to `items[0].product_id`).
- [ ] **`success_url`** lands on `/settings?subscription=success` and the extra query
  parameters Creem appends are harmless.
- [ ] **Webhook subscriptions and signing secret** set in Developers → Webhooks for
  `checkout.completed`, `subscription.*`, `refund.created`, `dispute.created`;
  test and live events go to separate webhook URLs per Creem's test-mode page;
  whether their signing secrets also differ is not stated, so confirm it and
  keep one secret per environment either way.
- [ ] **The hosted checkout locks the pre-filled email.** Creem identifies customers
  by email. If a buyer can change it, user A can pay with user V's email and A's own
  `jobber_user_id`: `checkout.completed` then links V's Creem customer to A (A reaches
  V's billing portal) and V's own later purchase hits the unique index and grants
  nothing (`ErrBillingAccountTaken`, `error` log). If the email is editable, compare
  `customer.email` with the user's email before linking.
- [ ] **`subscription.expired`.** The webhook reference says the status stays `active`
  and only `subscription.canceled` ends it; the refunds and introduction pages say
  expiry revokes access. We follow the former (the event is not acted on). Confirm
  with a real failed-charge run, and if `expired` means access is over, handle it.
- [ ] **Equal `updated_at`s.** An upgrade with an immediate proration charge can emit
  `subscription.update` and `subscription.paid` for one change; confirm they do not
  both carry the same `updated_at` with different content (a tie is dropped as
  superseded, except for a cancellation).
- [ ] **A downgrade** shows the expected `refund.created` warning and ends on the new
  plan.
- [ ] **Re-subscribing after a cancellation** (a new subscription id) replaces the old
  link and grants the plan.
- [ ] **The checkout and portal hosts.** The frontend only navigates to `creem.io` and
  `*.creem.io` (`BILLING_HOST_SUFFIX` in `fe/src/features/subscription/safeHttpsUrl.ts`).
  Confirm the real `checkout_url` and portal link hosts in the first test-mode
  purchase; a different (custom) domain would be refused with the generic checkout
  error until the constant is updated.
- [ ] **A user with no `subscriptions` row** that pays ends up on the plan (covered by
  tests; confirm once against a real account created before registration wrote a row).
- [ ] **Remove the production pin.** `.github/workflows/deploy-dev.yml` writes
  `FEATURE_PAYMENTS_ENABLED=false` and `FEATURE_BILLING_WEBHOOK_ENABLED=false` to the
  server and builds the frontend with `VITE_FEATURE_PAYMENTS=false`, whatever the
  secrets say, so production ships with payments off. Going live means reading
  those three from their secrets again (and setting the `CREEM_*` ones first, or
  the API refuses to boot).
- [ ] **Delete the stale GitHub Secrets**: all `FASTSPRING_*` and the earlier
  `PADDLE_*`, and add the five `CREEM_*` ones.
- [ ] **Legal pages** (Terms, Privacy, Refund) name Creem as Merchant of Record —
  confirm the wording is accurate for the refund policy actually in force.

## Known gaps, not built

- **Undoing a scheduled cancellation** from the app. The webhook reference says it
  can be reversed with a resume endpoint, but we have not confirmed its path or
  wired it; today the subscriber does that in the Creem portal, and a new checkout
  is refused with `ALREADY_SUBSCRIBED` in the meantime.
- **A second subscription bought outside the app** while one is live can be lost
  (see the link-conflict rules above). Recovery needs a new event for the second
  subscription (a plan change, the next renewal) or a manual fix of the row.
- **The `TRUSTED_PROXIES` default keeps gin's behaviour** (every proxy trusted, so
  `X-Forwarded-For` can be rotated to dodge per-IP limits, including the webhook
  route's). It is configurable now; set it in production once the proxy's address
  is known. It affects every rate limiter, not just billing.
- **`Incomplete`** appears on Creem's introduction page (23 hours to pay) but is not
  in the API's status enum or event list; an unrecognised status fails the event
  and is retried until Creem gives up.
