# Billing runs on FastSpring, with purchases linked to users server-side

Jobber's billing provider is **FastSpring** (Merchant of Record). Paddle is
removed entirely: there were no active or paying Paddle subscriptions at
cut-over, so no customer or payment data was migrated and no dual-provider
compatibility layer exists.

## The purchase → user chain

The only thing the browser is allowed to choose is a **plan name**. Everything
that identifies the buyer travels server-to-server:

1. `POST /api/v1/subscription/checkout-session` — authenticated. The backend
   reads the buyer's email and name from the **authenticated user record**, then
   calls the current Sessions API,
   [`POST /v2/checkouts/{checkoutPath}/sessions`](https://developer.fastspring.com/reference/createsession)
   ([overview](https://developer.fastspring.com/reference/sessions-overview)),
   with `live`, `locale`, `customer.billToContact`, `orderTags`, and
   `cart.lineItems[].productPath`. `customer.externalAccountId` is the merchant-owned
   key derived from the local user UUID (`jobber-<uuid-without-hyphens>`).
   The legacy `POST /sessions` endpoint is not used: FastSpring documents the v2
   endpoint as the one every new integration must build against.

   `locale` is a **two-letter language code**, not a regional tag: the endpoint
   documents plain codes (`en`, `ru`, …) and `en_US`/`ru_RU` are not among them.
   `billToContact` carries only name and email — the documented contact object
   has no `country` field at all (`country` is a *top-level* session field), so
   none is sent; adding one to the contact would be silently ignored.
2. FastSpring answers `201` with the session `id`, an ISO 8601 `expires`,
   `checkoutStatus` and `customer.accountId`. `checkoutStatus` is an **array** of
   statuses (`PRODUCTS_REQUIRED`, `READY_FOR_CHECKOUT`, `CONCLUDED`), so
   readiness is a membership test, never an equality check against a single
   string. The backend writes that account ID to
   `subscriptions.external_account_id` for the user *before* returning — plan and
   status are untouched, so an abandoned checkout grants nothing.

   The response's `checkoutUrls.webcheckoutUrl` is **not decoded and not
   returned**. It addresses the full-page hosted Web Checkout, which this
   integration no longer uses; keeping it in the DTO would only invite a
   redirect back into a flow that has none.
3. The frontend receives only `{session_id, expires_at}` and opens it in the
   **Store Builder Library popup** — `fastspring.builder.push({ checkout:
   session_id })`, the documented way to hand a Sessions API session to SBL.
   The session id is the whole cart: no product path, price or buyer detail is
   assembled in the browser, and there is no URL for anything to tamper with.
   A session is only returned when `checkoutStatus` *contains*
   `READY_FOR_CHECKOUT`; an absent or empty status list is not ready, and a
   session that never claimed to be payable is not opened.
4. [`subscription.activated`](https://developer.fastspring.com/reference/subscriptionactivated)
   arrives carrying `data.account.id`. The backend resolves the owner by, in
   order: the provider subscription ID, then that account ID, then — only if
   both miss — [`GET /accounts/{id}`](https://developer.fastspring.com/reference/retrieve-an-account)
   to read back `lookup.custom` and decode the user UUID from it.

   > ⚠️ **Unverified assumption — must be confirmed with a real test-mode
   > purchase.** Step 4's last hop assumes the session's
   > `customer.externalAccountId` is what FastSpring stores as the account's
   > `lookup.custom`. The two fields are documented separately and nothing in the
   > reference states that the first populates the second; the code was written
   > against that reading, not against an observed payload. Until a test-mode
   > purchase confirms it (see the go-live checklist below), treat this path as
   > *unproven*. It is only a third-line fallback — the subscription ID and the
   > account ID recorded at session creation resolve the normal case — so if the
   > assumption turns out to be wrong, an unresolvable event fails and is retried
   > rather than granting access to the wrong user.

Every link in that chain is a value Jobber itself wrote through an authenticated
server-to-server call. A `user_id` posted from client JavaScript is never read;
the request DTO carries a plan and nothing else.

## One subscriber, one subscription

`subscriptions` holds a **single** `external_subscription_id` per user. A second
checkout would overwrite it: the first subscription would keep billing at
FastSpring with nothing in Jobber pointing at it, so neither the user nor the app
could cancel it.

So a user who already holds a provider subscription cannot start another one.
`CreateCheckoutSession` reads their subscription before it calls FastSpring and
answers `ErrAlreadySubscribed` → **HTTP 409 `ALREADY_SUBSCRIBED`**. `active`,
`past_due`, `paused` and a scheduled cancellation (`active` + `cancel_at`) all
still bill, so all four are refused. Only two things allow a fresh checkout: no
provider subscription ID at all, and status `cancelled` — the provider has ended
that one, so buying again is the only way back to a paid plan. A database error
is never swallowed into "go ahead": an unreadable subscription fails the
checkout rather than bypassing the guard.

The UI enforces the same rule one step earlier, so a subscriber never sees the
409: the pricing and upgrade modals route a paying user to
`POST /subscription/change-plan` instead of a checkout. Going *back* to free is
not a plan change but a cancellation, so the free card offers no CTA to a
subscriber at all — a dead button would only look broken. The 409 remains as
defence in depth for any other client.

`order.completed` is acknowledged but deliberately **not** acted on. Entitlement
comes from the subscription lifecycle events alone, which avoids a second grant
racing `subscription.activated`. An `orderTags` value is still sent at session
creation as a diagnostic breadcrumb, but no code reads it: tags are not part of
the documented subscription event payload, so relying on them would be guessing.

## One config value governs both halves of the checkout

`FASTSPRING_CHECKOUT_PATH` names the dashboard checkout. The endpoint documents
`checkoutPath` as exactly `storefront-id/checkout-id`, so the value must be
**exactly two segments** — a single segment, three or more, or a leading/trailing
slash is a misconfiguration and is refused rather than trimmed into something the
operator did not write. Each segment is then validated against
`[A-Za-z0-9][A-Za-z0-9._-]*` before it is put in a URL, so no configured value
can traverse into a different API endpoint.

A third check is specific to the popup: the checkout id must start with
`popup-`. FastSpring generates that prefix for a checkout created as a popup
checkout, and SBL keys its own behaviour off it — it draws the checkout as an
on-page iframe only when the URL it is about to open matches `…/popup-…` (or
`…/embedded-…`), and otherwise assigns that URL to `window.location`. A leftover
full-page Web Checkout path such as `fluxlab/jobber-checkout` passes every other
check and then navigates a real buyer off Jobber — the exact behaviour this
design removes — so it fails the boot instead. The configured value is
`fluxlab/popup-jobber` — generically, `store/popup-name`.

The **popup storefront** the browser loads SBL against is *derived* from that one
value plus `FASTSPRING_ENVIRONMENT`, never configured separately:

| environment | `data-storefront` |
| --- | --- |
| `test` | `fluxlab.test.onfastspring.com/popup-jobber` |
| `live` | `fluxlab.onfastspring.com/popup-jobber` |

Deriving it is the point: "which checkout the API creates a session against" and
"which storefront the browser opens" cannot name two different checkouts, and a
test deployment cannot be pointed at the live storefront by editing one variable
and forgetting the other. It reaches the browser through
`GET /subscription/checkout-config`, which carries the provider name, the
environment, the storefront and the purchasable plan names — and no credentials,
no API keys and no catalog product paths.

The frontend does not trust it blindly either. Before the storefront goes into a
script tag it must be a plain `host/path` under `onfastspring.com`, its test
marker must agree with the environment the same response reports, *and* its
checkout id must carry the `popup-` prefix — the last one because a storefront
without it makes SBL navigate the page instead of drawing a frame, so the
frontend refuses rather than inherits a full-page redirect. The SBL script
`src` is a pinned constant (`1.0.9`, the build the dashboard's own snippet
names), and so are the two global callback names — nothing script-related is ever
taken from a backend response.

All checks run before any network call.

## Why `subscription.canceled` keeps access

In FastSpring, `canceled` means *cancellation scheduled*: the payload still has
`active: true` and carries a `deactivationDate`. Only `subscription.deactivated`
ends access. So `canceled` maps to internal status `active` with `cancel_at` set
to the deactivation date, and `deactivated` maps to `cancelled` + `free`.

`subscription.paused` and `subscription.resumed` are handled too: a pause maps to
internal status `paused`, which keeps the purchased plan on the row but drops the
*effective* plan to free, so paid quotas stop while billing is suspended. A
resume returns the row to `active` and the quotas with it.

### Status mapping is an ordered set of rules

The order is the contract, because the payload's `active` flag alone is
ambiguous — FastSpring reports a *paused* subscription as `active: false` too:

1. `state: "deactivated"` → `cancelled`. Nothing outranks it: no dunning notice,
   no resume and no replay can revive a subscription the provider has ended.
2. `subscription.paused` → `paused`, **even when `active` is false**. Reading
   that flag as a cancellation would rewrite the row to the free plan, and the
   resume would then have no purchase left to give back.
3. `subscription.resumed` → `active`, even when `active` has not caught up yet.
4. Only then does a generic `active: false` mean `cancelled`.
5. `subscription.payment.overdue` / `subscription.charge.failed` on an otherwise
   active subscription → `past_due`, so the dunning period is visible.
6. Otherwise the payload's `state` decides; an unrecognised state fails the event
   rather than guessing.

Cancellation therefore uses
[`DELETE /subscriptions/{id}?billingPeriod=1`](https://developer.fastspring.com/reference/cancel-a-subscription),
which is end-of-period. `billingPeriod=0` would cancel immediately and is not
used by any user-facing path.

## Safety rules the code enforces

- **Signature first.** `X-FS-Signature` is `base64(HMAC-SHA256(rawBody))`
  ([Message Security](https://developer.fastspring.com/docs/message-security)).
  It is verified against the raw body before the payload is parsed or any query
  runs, so a forged or unsigned request cannot reach the database.
- **Environment guard.** Each event carries `live`. An event from the other mode
  is acknowledged (retrying could never make it processable) but never applied.
  Acknowledging *loses* it, and the only cause is a deployment pointed at the
  wrong environment, so this one skip is logged at `warn` while every other skip
  stays at `info`.
- **Someone else's customer is skipped, not retried.** The FluxLab store sells
  more than Jobber, so the other products' subscription lifecycle events arrive
  on this endpoint too. When neither local ID resolves an event, the account's
  `lookup.custom` decides: no `jobber-` key means the account is not ours, and
  the event is acknowledged and dropped — no retry could ever make it
  resolvable. The two neighbouring failures stay retryable on purpose: an
  unreachable account API says nothing about ownership, and a `jobber-` key
  whose user row is missing is a link to repair, not a foreign purchase.
- **Unknown product never pays.** A `product` path that matches neither
  configured plan fails the event, so it is retried after the catalog is fixed
  rather than silently granting or silently dropping a paid subscription.
  Revocation (`deactivated`) does not consult the product at all, so a catalog
  mistake can never block ending access.
- **One atomic write per event.** The `webhook_events` claim and the
  `subscriptions` write happen in a *single* statement (a CTE), so they roll back
  together. A separate claim would survive a failed write, and the provider's
  retry would then be acknowledged as a duplicate while the subscriber stayed on
  the old plan. The statement reports `applied`, `duplicate` or `superseded`.
- **Replays cannot resurrect.** Automatic retries repeat the event ID and are
  caught by the claim. Manual resends get a *fresh* ID **and a fresh envelope
  `created`**, so ordering uses the payload's own `data.changed` — the provider's
  timestamp for the state change, which a resend does not move. `created` is only
  a fallback for payloads that carry no `changed` (the charge events).
  `subscriptions.last_event_at` holds that value, and the ordering guard lives in
  the UPDATE's `WHERE` clause rather than in a prior read, so two deliveries
  racing on one row cannot interleave into the older state.
- **The ordering guard is strict.** The condition is
  `EXCLUDED.last_event_at > subscriptions.last_event_at`, not `>=`. An event
  carrying the *same* `data.changed` as the applied state describes a change that
  has already been accounted for, so re-applying it can only undo a correct
  write. The realistic pair — `subscription.charge.completed` and the
  `subscription.updated` it triggers, both stamped with one `changed` — normalise
  to the same state, so applying only the first loses nothing: first writer wins,
  the second is recorded as `processed`/`superseded` and stops being redelivered.
  A genuine later change always carries a later `changed`, so nothing real is
  dropped.
- **Every actionable event must be identifiable.** An actionable event with no
  `id` cannot be de-duplicated, so it is treated as malformed: nothing is
  written and the provider redelivers. No event is ever acknowledged as an empty
  ID, which FastSpring would not recognise.
- **Partial batches.** A batch acknowledges with `200` when every event is
  processed, or `202` listing the processed IDs one per line when only some are
  ([ack contract](https://developer.fastspring.com/docs/processed-and-unprocessed-webhook-events)).
  A failed event leaves no claim behind at all, so its retry gets a clean run
  while the rest of the batch stays acknowledged.

## Checkout language

`locale` is a two-letter FastSpring language code. Jobber maps its own UI
locales onto the codes the storefront can actually render:

| Jobber locale | FastSpring `locale` |
| --- | --- |
| `ru` | `ru` |
| `ua`, `uk` | `ru` |
| everything else (incl. `en`, empty, unknown) | `en` |

**Ukrainian falls back to Russian**, and that is a deliberate, imperfect choice:
FastSpring's documented checkout language set has no Ukrainian entry
(`ar cs da de es en fi fr hr it iw ja ko nl no pl pt ru sk sv tr zh`), so `uk`
cannot be requested at all. The fallback follows FastSpring's own *default
language for Ukraine*, which is a **store-level dashboard setting, not an API
guarantee** — it is on the verification list below. If the store's Ukraine
default turns out to be English, the `ua`/`uk` arm of `checkoutLocale` becomes
`localeEnglish` and nothing else changes.

Regional tags (`en_US`, `ru_RU`, `uk_UA`) are never sent: they are not valid
values for this field.

## Must be confirmed in the FastSpring dashboard (test mode) before go-live

These are the places where the code encodes a reading of the docs or a dashboard
setting that no unit test can prove. Each needs one real **test-mode purchase**:

- [ ] **`externalAccountId` → `lookup.custom`.** Create a session, complete a
  test purchase, then `GET /accounts/{id}` and check `lookup.custom` really
  holds the `jobber-<uuid>` key the session sent. This is the assumption behind
  the third-line webhook fallback (step 4 above) and it is **not** stated in the
  reference. If it does not hold, that fallback is dead code and the resolution
  chain must rely on the subscription/account IDs alone.
- [ ] **Default checkout language for Ukraine.** Open a checkout with
  `locale: "ru"` from a Ukrainian context and confirm the storefront renders as
  expected — FastSpring's per-country default language is a store setting.
- [ ] **`checkoutStatus` shape on a live-ish response.** Confirm the created
  session answers with an array containing `READY_FOR_CHECKOUT`.
- [ ] **Popup checkout path.** Confirm the configured
  `storefront-id/checkout-id` matches the dashboard's **popup** checkout
  (`fluxlab/popup-jobber`), that it is two segments, and that the derived
  storefront (`fluxlab.test.onfastspring.com/popup-jobber`) is the one the
  dashboard shows.
- [ ] **Live storefront is online.** The live store is currently *Offline*, so
  `fluxlab.onfastspring.com/popup-jobber` cannot be exercised until it is
  published — the live derivation is only verified by that.
- [ ] **Digital wallets in the popup.** Decide whether Apple Pay / Google Pay
  buttons are expected. SBL already sets `allow="payment"` on its iframe, so the
  remaining half is `Permissions-Policy: payment` naming the storefront origin.
  Confirm in test mode whether wallets are wanted before widening the header.
- [ ] **Account Management Portal host.** The portal URL from
  `GET /accounts/{id}/authenticate` is validated as absolute HTTPS and nothing
  more. Its host is deliberately *not* whitelisted: a FastSpring storefront can
  run on a merchant custom domain and no documented contract pins the host to
  `*.onfastspring.com`. Confirm what the endpoint actually returns for this
  store; only then is narrowing it to a host family safe.
- [ ] **Webhook subscriptions and the HMAC secret**, per the Consequences below.

## Two independent kill switches

`FEATURE_PAYMENTS_ENABLED` gates **new purchases and plan management**;
`FEATURE_BILLING_WEBHOOK_ENABLED` gates **webhook ingestion**. They are separate
so closing the checkout does not stop renewals, cancellations and deactivations
from being recorded for people who already paid — set the webhook flag to `true`
explicitly for that.

Left unset, the webhook flag **follows `FEATURE_PAYMENTS_ENABLED`**. That default
exists because webhook ingestion without `FASTSPRING_WEBHOOK_SECRET` is worse
than useless: every delivery is rejected, so the server looks healthy while
silently dropping every lifecycle event. `Load()` therefore *refuses to start*
when ingestion is on and the secret is missing, and following the payments flag
is what keeps a local billing-off stack booting with no FastSpring config at all.

Two more things fail the boot rather than a real buyer's click: missing API
credentials when payments are on, and a `FASTSPRING_CHECKOUT_PATH` that does not
pass the same `EscapeCheckoutPath` whitelist the API client applies.

## Consequences

- Both subscribed webhook types and the checkout id are dashboard settings, so
  they belong on the go-live checklist: subscribe to `subscription.activated`,
  `.updated`, `.canceled`, `.uncanceled`, `.deactivated`, `.paused`, `.resumed`,
  `.charge.completed`, `.charge.failed`, `.payment.overdue` and `order.completed`.
- Checkout runs on our own origin, so the CSP has to make room for it. Five
  scoped allowances, each one observed in an actual popup run rather than
  guessed: `script-src https://sbl.onfastspring.com` (the SBL file),
  `style-src https://sbl.onfastspring.com` (the `fastspring.css` SBL appends from
  its own directory on start, which the popup chrome needs),
  `img-src https://sbl.onfastspring.com` (the popup's loading spinner),
  `frame-src https://*.onfastspring.com` (the storefront iframe SBL renders) and
  `connect-src https://*.onfastspring.com` (SBL's `builder` and `finalize` calls
  to the storefront).
  Scripts get neither `'unsafe-inline'` nor `'unsafe-eval'`, and cdnjs stays out:
  SBL only reaches for it to render DynaCart markup, which Jobber has none of. `form-action` keeps
  its existing `https://*.onfastspring.com` allowance: nothing submits a form
  cross-origin now that the redirect is gone, but SBL is third-party code in our
  documents and the cost of being wrong is a checkout that only breaks in
  production — the allowance stays scoped to an origin family already trusted for
  script and frame. `Permissions-Policy` is unchanged, which leaves `payment` at
  its default `self`: the FastSpring iframe cannot use the Payment Request API,
  so digital-wallet buttons will not render. SBL does set `allow="payment"` on
  that iframe, so this header is the only thing withholding the feature. Card
  payment is unaffected, and widening it is a decision about wallets on the
  checklist above rather than a fix this change needs.
- The script is inserted **once**, lazily, by the checkout hook — a visitor who
  never clicks Upgrade never downloads it, and payments being switched off means
  it is never requested at all.
- Nothing navigates, so a completed purchase produces no page load. The popup's
  close callback dispatches a same-page event that the layout listens for and
  turns into the same polling it would have run after a redirect. A reload while
  the popup is open destroys the popup with the page, so the `sessionStorage`
  baseline is still read at startup for exactly that case — and the
  `?subscription=success` parameter is still honoured, since it costs nothing.
- The close callback is **not** evidence of payment. All it decides is whether to
  start polling; the success modal still waits for the backend to report a
  strictly higher plan than the recorded baseline, and that only moves on a
  webhook. A popup closed without an order clears the baseline instead, so no
  overlay and no success state appears. The callback payload is never logged and
  never read beyond "is it there at all" — SBL passes the popup's order
  references on a concluded order and `null` on a plain close, so presence is the
  documented signal and its fields are not inspected.
- A buyer who reloaded mid-purchase comes back to a waiting overlay, so it is
  dismissable: one click clears the baseline and stops the polling immediately
  instead of holding the app hostage until the five-minute timeout. A `pageshow`
  handler covers a bfcache restore (returning from the Account Management
  Portal), where no remount would otherwise clear it.
- Invoices, receipts, the payment method and refund requests all live on the
  provider's side, so the manage-subscription modal links out to the Account
  Management Portal. The URL only exists after an async round-trip, by which
  point `window.open` is severed from the user gesture and gets blocked as a
  popup — so the portal is opened by navigating the current tab.
- `subscriptions.external_account_id` has a partial UNIQUE index: one FastSpring
  account resolves to exactly one user. A collision fails the checkout rather
  than risking a mis-grant.
