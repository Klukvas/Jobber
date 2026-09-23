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

   **A first-time buyer has no account yet.** FastSpring mints it *during*
   checkout, so `customer.accountId` comes back empty and there is nothing to
   link. That is the normal case for every new subscriber, not an error — step 4
   is what covers it.

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
   arrives carrying the account (`data.account`, an object on most events and a
   bare ID string on others) and the order's `data.tags`. The backend resolves
   the owner by, in order: the provider subscription ID, then the **proven
   `jobber_user_id` order tag** (see *The order tag is a claim, not an
   identifier* below), then the account ID, then —
   only if all three miss — [`GET /accounts/{id}`](https://developer.fastspring.com/reference/retrieve-an-account)
   to read back `lookup.custom` and decode the user UUID from it. Whichever hop
   wins, the event's account ID *and* subscription ID are written to the row, so
   every later event resolves on the first hop.

   **Why the tag outranks the account.** Once it has passed its MAC, the tag is
   the strongest evidence in the payload and the only evidence bound to *this*
   order: it names the user whose authenticated session created this checkout.
   The account is weaker on both counts — FastSpring records it per buyer
   contact rather than per purchase and reuses one account across purchases made
   with the same contact, so two local users who check out with one email end up
   behind one account and the account hop would hand one of them the other's
   purchase. The subscription ID stays above the tag because it is exact, and it
   is what every event after the first resolves on.

   The tag is also read before *any* account handling, including the guard for
   an event carrying no account at all. That guard used to sit above it, which
   made the one hop designed to link a first purchase unreachable for exactly
   the payloads most likely to need it — and left them retrying until FastSpring
   gave up. An event that still identifies nobody stays retryable rather than
   being acknowledged away as foreign: a payload we failed to read is our bug,
   not somebody else's customer.

   > ✅ **Settled by a test-mode purchase (order for `jobber-enterprise`,
   > `subscription.activated` `EVWBPALB…`).** The assumption the first version of
   > this design rested on — that a session's `customer.externalAccountId`
   > becomes the account's `lookup.custom` — is **false for this store**.
   > `GET /accounts/{id}` for the account that purchase created returns a
   > `lookup` holding only `global`; there is no `custom` key. Combined with the
   > empty `customer.accountId` in step 2, that left a first purchase with *no*
   > resolvable identifier at all: the event was classified as another product's
   > and acknowledged away, and the buyer stayed on the free plan.
   >
   > The same signed payload did carry `data.tags.jobber_user_id`, which is why
   > the order tag is now a resolution hop rather than a breadcrumb. The
   > `lookup.custom` hop is kept below it: it costs nothing when the tag already
   > resolved, and it still works if FastSpring populates the field later.

Three of those four hops read a value Jobber itself wrote through an
authenticated server-to-server call and then stored: the subscription ID, the
account ID, and the account's custom lookup key. The other — the order tag — is
the one value that arrives *inside* the event, and it is trusted only because of
the proof described below. A `user_id` posted from client JavaScript is never
read; the request DTO carries a plan and nothing else.

## The order tag is a claim, not an identifier

**Order tags are not a server-only channel.** The Store Builder Library exposes
[`fastspring.builder.tag()`](https://developer.fastspring.com/docs/store-builder-library-sbl),
so any visitor to the shared FluxLab storefront can attach arbitrary tags to
their *own* order, and FastSpring echoes them into the webhook payload exactly
like the ones Jobber's session-creation call wrote. The webhook HMAC proves that
**FastSpring sent the event** — it says nothing about **who authored a tag inside
it**.

Reading `jobber_user_id` on its own would therefore be a privilege-escalation
path: load the storefront directly, tag the order with a known victim's UUID, buy
`jobber-enterprise`, and the victim's row is repointed at the attacker's
subscription — the victim loses their own checkout (one subscription per user)
and the account-management portal authenticates them into the attacker's billing
account. A product allowlist and a UUID format check bound that attack; they do
not remove it.

So the ID travels with a **proof**: `jobber_user_proof`, an HMAC-SHA256 over a
domain- and version-separated message containing that same canonical UUID,
minted at session creation and verified in constant time when the event comes
back. An unauthenticated storefront visitor can write the ID; they cannot write a
MAC they have no key for, and moving a proof onto a different UUID invalidates it.

The MAC key is the **FastSpring webhook secret**, deliberately:

- it is already the root of trust for this path. Forging a proof needs the same
  secret that would let an attacker forge an entire signed event, so reusing it
  hands an attacker nothing they did not already have;
- it exists in exactly the deployments where the proof matters — the secret is
  required whenever webhook ingestion is enabled, and with it absent no event is
  accepted at all, so there is no window where a claim is read but unprovable by
  configuration;
- it needs no new environment variable, no new secret to leak, and no migration.

Domain separation (`jobber.fastspring.order-tag`, version `v1`, NUL-separated)
keeps the two uses of that key apart in the direction that matters: FastSpring
signs only its own event batches, and a JSON object is never a message beginning
with that domain string — so no captured body signature is also a valid proof.

The reverse is not a property of the construction, and saying so is cheaper than
pretending otherwise. A user's own proof is a MAC under the same secret, so its
owner can re-encode it as an `X-FS-Signature` and POST the one body it matches:
the raw domain message. That request passes signature verification and then dies
in parsing — the body is not a JSON event batch, so no event ever exists to
apply. It buys an attacker a 400 on their own request, and no other body, because
signing one would still need the secret.

Two consequences worth stating plainly:

- **Rotation.** A proof minted before the webhook secret changes will not verify
  after it, so a checkout in flight across a rotation loses its tag hop. That
  window is one checkout-session lifetime — the same window in which in-flight
  deliveries already fail their signature — and it fails loudly
  (`ErrUnprovenOrderTag`, logged at `warn`), never by resolving the wrong user.
- **No expiry, no nonce.** The proof is deterministic per user on purpose. It
  authorises exactly one thing — "this order belongs to user X" — and the only
  party who can *obtain a fresh* proof for X is X, through an authenticated call.
  Replay by its owner attributes their own purchase to themselves, which is what
  the tag is *for*; the one-subscription guard below is what stops that from
  repointing a subscription they are already paying for. An expiry would buy
  nothing against that and would start rejecting legitimate late deliveries — a
  webhook retried for hours, or an activation that lands long after checkout.

  One caveat worth stating rather than glossing: the proof is also *stored at
  FastSpring*, on the order, so anyone with merchant read access to the
  dashboard or the API can read one for a user who has checked out. That is not
  a hole in the construction — such a party already holds the webhook secret
  the proof is keyed with, and everything else it protects — but "only X can
  obtain a proof for X" is true of the minting path, not of the whole system.

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

That check is the first line of the rule, not the rule itself. It runs before
FastSpring is called, so nothing stops a user from opening two checkouts while
they are still free — both reads see a row with nothing to protect — and then
paying for both. The second activation resolves through the provider account ID,
a hop no order tag is involved in, and would overwrite the identifier the first
one wrote.

So the invariant is enforced where the write happens. `ApplySubscriptionEvent`
opens a transaction, locks the user's row (`SELECT … FOR UPDATE`), and refuses
any event that would replace a non-null `external_subscription_id` with a
different one while the stored status is not `cancelled` — reporting
`WebhookLinkConflict` → `ErrSubscriptionLinkConflict`, logged at `warn`. Three
things stay allowed: the first link, when nothing is stored; any number of
lifecycle events for the subscription the row already names; and a replacement
once the provider has ended the old one. The same condition is repeated in the
write's own `WHERE` clause, so it is re-evaluated against the row version a
concurrent writer committed rather than against the read that preceded it.

A refused event claims nothing and writes nothing — the transaction rolls back
whole — so once the stale link is genuinely ended, a resend from the dashboard
can still land. It is acknowledged rather than retried, because no redelivery
makes a second subscription fit one row, and it is reported as its own outcome
rather than as a routine supersede. That distinction is the point of the `warn`:
somebody may be paying for two subscriptions with only one of them cancellable
from Jobber, and a human has to decide which one to end.

The UI enforces the same rule one step earlier, so a subscriber never sees the
409: the pricing and upgrade modals route a paying user to
`POST /subscription/change-plan` instead of a checkout. Going *back* to free is
not a plan change but a cancellation, so the free card offers no CTA to a
subscriber at all — a dead button would only look broken. The 409 remains as
defence in depth for any other client.

`return.created` is subscribed to and deliberately **not** acted on either, but
for a different reason, and it is reported as its own outcome rather than as a
routine skip. The payload names an *order*, not a subscription, and a refund is
not a cancellation: a partial refund leaves the subscription billing normally,
so revoking access on one would take a paid plan from somebody who still has it.
Ending a subscription over a refund is a merchant decision made in the
dashboard, and it arrives here afterwards as the deactivation it really is. What
it must not be is invisible — money left the account and nothing in the app
moved — so it logs at `warn` next to the other four non-routine skips.

`order.completed` is acknowledged but deliberately **not** acted on. Entitlement
comes from the subscription lifecycle events alone, which avoids a second grant
racing `subscription.activated`. It stays a *routine* skip, which is what makes
the refund's own outcome worth anything. The `orderTags` sent at session creation are
read from the *subscription* events instead, where FastSpring echoes them back as
`data.tags` — so nothing has to act on the order to know whose purchase it was.

The tag path carries the same guard one step earlier, and keeps it for the
diagnosis rather than for the invariant: a buyer can replay their own proof onto
a storefront purchase Jobber never brokered, and a tag naming a user who already
holds a live provider subscription is refused as `ErrTaggedOwnerConflict` — which
names the suspicious input, an order tag, instead of leaving it to look like
Jobber's own flows colliding. It is a read taken before a separate write, so it
cannot be the last word; the atomic guard above is, on that hop as on every
other.

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

- **The endpoint is public, so it is rate limited.** It is the only route with
  no auth middleware in front of it: the HMAC is the authentication, but
  verifying it means reading and hashing the whole body first, so an unsigned
  flood still buys real work. The limit sits far above anything FastSpring
  produces and fails open on a Redis error, deliberately — a throttled delivery
  is a lifecycle event that has to come back through the retry schedule, and one
  that never comes back is a subscriber on the wrong plan. The body cap stays at
  1 MB for the same reason: a tighter one would start refusing real batches.
- **Signature first.** `X-FS-Signature` is `base64(HMAC-SHA256(rawBody))`
  ([Message Security](https://developer.fastspring.com/docs/message-security)).
  It is verified against the raw body before the payload is parsed or any query
  runs, so a forged or unsigned request cannot reach the database.
- **Environment guard.** Each event carries `live`. An event from the other mode
  is acknowledged (retrying could never make it processable) but never applied.
  Acknowledging *loses* it, and the only cause is a deployment pointed at the
  wrong environment, so it is logged at `warn`. Two other skips are — the
  unproven order tag and the tagged-owner conflict below; every routine skip
  stays at `info`.
- **Someone else's customer is skipped, not retried.** The FluxLab store sells
  more than Jobber, so the other products' subscription lifecycle events arrive
  on this endpoint too. When neither local ID resolves an event, the order tag
  and then the account's `lookup.custom` decide: no `jobber_user_id` tag at all
  *and* no `jobber-` key means the purchase is not ours, and the event is
  acknowledged and dropped — no retry could ever make it resolvable. The two
  neighbouring failures stay retryable on purpose: an unreachable account API
  says nothing about ownership, and a Jobber identifier whose user row is missing
  is a link to repair, not a foreign purchase.
- **The order tag is trusted under three conditions, and only three.** It arrives
  inside the HMAC-verified body, but that proves only that FastSpring sent the
  event — the storefront's own `fastspring.builder.tag()` lets a visitor write
  tags too, so on its own it authorises nothing:
  1. **The product must be Jobber's.** A purchase of another FluxLab product can
     never claim a Jobber user, however its order is tagged — that is what keeps
     a shared store from being a privilege-escalation path. Checked first, so a
     foreign product's tags are not read at all.
  2. **The claim must carry a valid `jobber_user_proof`** for the canonical UUID
     it names — the MAC described above, compared in constant time. A missing,
     malformed, truncated, wrong-version, wrong-user or wrong-secret proof
     resolves nothing. Because the product is one of ours, this is an integrity
     anomaly rather than a routine miss: it is reported as `ErrUnprovenOrderTag`,
     acknowledged (no redelivery could make an unprovable claim provable) and
     logged at `warn`. It does **not** fall through to another hop — a payload
     Jobber has reason to distrust is not resolved by other means.
     An order carrying *no* user tag makes no claim at all and is a different
     thing: it simply falls through to the lookup key, which is where every
     foreign order in the shared store ends up.
  3. **The user it names must not already hold a live provider subscription.** A
     tag is written once per checkout and a paying user cannot start a second
     one, so a tag that contradicts the row is refused rather than applied:
     honouring it would overwrite the single `external_subscription_id` and
     strand the subscription that user actually pays for. This guard survives the
     proof — its owner can replay it onto a storefront purchase Jobber never
     brokered — so it is what keeps a proven tag from repointing a paying row.
     The event is acknowledged (no retry can resolve a contradiction) and logged
     at `warn`, next to the environment mismatch, because it is a skip that
     should never happen on its own.
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

## The webhook is a push channel, so there is also a pull channel

Every guard above assumes the event arrives. A push channel does not guarantee
that: FastSpring retries a failed delivery for **up to 7 days and at most 12
attempts**, then marks it permanently failed and never sends it again. A
deployment that was down, unreachable, or holding a stale secret for longer than
that window ends up with subscription rows that nothing will ever correct — a
buyer on the free plan they paid to leave, and a `warn` line in a log as the
only trace.

So a background sweep pulls what the push channel lost, which is
[FastSpring's own documented remedy](https://developer.fastspring.com/reference/processed-and-unprocessed-webhook-events):
[`GET /events/unprocessed`](https://developer.fastspring.com/reference/list-all-unprocessed-events)
still lists every event with no acknowledgement, and
[`POST /events/{id}`](https://developer.fastspring.com/reference/update-an-event)
settles one. The sweep runs every 6 hours over a 14-day window — double the
retry schedule, so an event only just given up on cannot fall between two
sweeps — plus once shortly after boot, because a restart is exactly what
follows the outage this exists for.

Three things make it safe to reuse the ingestion pipeline verbatim:

- **The trust model is unchanged.** A webhook body is trusted because its HMAC
  proves FastSpring sent it; this response is trusted because Jobber fetched it
  over TLS with its own API credentials. Both establish the same one fact — the
  provider sent this — and neither says anything about who authored a tag
  inside it. Every check that reads the event's *contents* is the same code:
  the environment guard, the product allowlist, the order-tag proof, the link
  guard and the event claim all run unchanged.
- **Applying is idempotent.** A sweep overlapping a live delivery is recognised
  by the event claim and reported as a duplicate, so nothing is granted twice.
- **Only settled events are acknowledged.** Applied, duplicate, superseded and
  deliberately-skipped events are reported back; a failed one is left
  unacknowledged on purpose, because that is precisely what makes the next
  sweep retry it. An acknowledgement that itself fails costs one duplicate on
  the next sweep and nothing else.

It is gated on `FEATURE_BILLING_WEBHOOK_ENABLED` rather than on payments: that
flag is what guarantees the secret every order-tag proof is verified against,
and a deployment that deliberately ingests nothing must not start pulling. With
no API credentials or no secret the sweep refuses to run at all rather than
dragging in purchases it could not prove.

**This replaces polling each subscription for state drift, and deliberately.**
Every state change *is* an event, and an event that is not acknowledged is
replayed — so a second mechanism comparing rows against
`GET /subscriptions/{id}` would re-derive what this already recovers, with its
own drift to maintain. What it does not cover is an event the pipeline
acknowledged and dropped on purpose (an environment mismatch, an unproven tag,
a link conflict). Those are the four `warn` lines above, they require a human
decision anyway, and no amount of polling would make them safe to apply
automatically.

## Checkout language

`locale` is a two-letter FastSpring language code. Jobber maps its own UI
locales onto the codes the storefront can actually render:

| Jobber locale | FastSpring `locale` |
| --- | --- |
| `ru` | `ru` |
| everything else (incl. `en`, `ua`, `uk`, empty, unknown) | `en` |

**Ukrainian falls back to English.** FastSpring's documented checkout language
set has no Ukrainian entry
(`ar cs da de es en fi fr hr it iw ja ko nl no pl pt ru sk sv tr zh`), so `uk`
cannot be requested at all and a Ukrainian buyer will not see a Ukrainian
checkout whatever this maps to. The only question is which renderable language
they see instead.

An earlier version answered Russian, following FastSpring's own regional
default. That is the wrong thing to ship for this audience: a payment form in
Russian is a reason to close the tab, and a checkout nobody completes costs far
more than one read in a second language. English is what the rest of the app
already falls back to, so it is what the checkout falls back to.

Regional tags (`en_US`, `ru_RU`, `uk_UA`) are never sent: they are not valid
values for this field.

## Must be confirmed in the FastSpring dashboard (test mode) before go-live

These are the places where the code encodes a reading of the docs or a dashboard
setting that no unit test can prove. Each needs one real **test-mode purchase**:

- [x] **`externalAccountId` → `lookup.custom` — checked, and it does not hold.**
  A test-mode purchase of `jobber-enterprise` produced an account whose
  `GET /accounts/{id}` returns a `lookup` with `global` only. The session's
  `externalAccountId` does **not** become `lookup.custom` for this store, so that
  hop resolves nothing in practice and the `jobber_user_id` order tag carries the
  first purchase instead (step 4). The hop is kept below the tag rather than
  deleted: it costs nothing once the tag has resolved, and it is the fallback if
  FastSpring ever does populate the field.
- [ ] **The order-tag proof survives a real round trip.** Complete one test-mode
  purchase through the app's own checkout and confirm the
  `subscription.activated` payload carries **both** `data.tags.jobber_user_id`
  and `data.tags.jobber_user_proof`, and that the buyer lands on the paid plan.
  FastSpring's tag length/charset limits are not stated in the reference, and a
  silently truncated proof would look exactly like a forged one: `warn`
  "order tag names a user it cannot prove", plan unchanged. The tag hop is what
  links every first purchase, so this one needs the real payload.
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

`FASTSPRING_WEBHOOK_SECRET` also keys the order-tag proof, so the two flags stay
consistent there too: with ingestion off there is no secret to mint a proof with,
and the checkout sends no user tag at all rather than an unverifiable one — a bare
ID on the wire would look like provenance without being any. Nothing is lost by
that, because with ingestion off no event would ever read it. Rotating the secret
is still a coordinated dashboard-plus-deploy change: checkouts in flight across
the rotation lose their tag hop and are reported at `warn`, never resolved to the
wrong user.

## Consequences

- Both subscribed webhook types and the checkout id are dashboard settings, so
  they belong on the go-live checklist: subscribe to `subscription.activated`,
  `.updated`, `.canceled`, `.uncanceled`, `.deactivated`, `.paused`, `.resumed`,
  `.charge.completed`, `.charge.failed`, `.payment.overdue`, `order.completed`
  and `return.created`. The last one changes nothing on its own — see above —
  but without it a refund or chargeback never reaches the logs at all.
- The API credentials need **events read and write** for reconciliation to work,
  on top of what checkout and subscription management already use. Without them
  the sweep logs `ErrNotConfigured` and recovers nothing, while the push channel
  keeps working — so a missing scope is quiet, and belongs on the checklist.
- `webhook_events` claims are expired after 90 days by the hourly cleanup job.
  The claim exists to recognise a redelivery, nothing can be redelivered past
  the reconciliation window, and the lifecycle ordering guard refuses anything
  not strictly newer even if one were forgotten early.
- The API client keeps **two** circuit breakers, split by who is waiting on the
  call. The shared store means other products' webhook deliveries drive account
  read-backs here, and with one breaker three of those failing in a row refused
  a real buyer's checkout for the next thirty seconds.
- A `external_account_id` collision answers **409 `BILLING_ACCOUNT_TAKEN`**, not
  a bare 500. It means two local users are behind one provider account — which
  happens when they check out with the same email — and refusing is the only
  alternative to mis-granting a purchase. Nobody can untangle it from inside the
  app, so the buyer is told to contact support and support gets a `warn`.
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
