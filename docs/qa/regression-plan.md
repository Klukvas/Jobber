# Jobber UI regression plan (production, jobber-app.com)

Target: https://jobber-app.com. Browser: Chromium (Playwright), desktop viewport 1366x768, mobile 390x844 where stated.
Scope: 100 manual-style UI cases across 10 blocks, one tester (agent) per block, one dedicated test account per block.
Out of scope: payments/checkout (feature-flagged off in production), Google Calendar (flag off), Chrome extension, backend/API-only behavior.

Production facts that shape expectations (from code on `main`):
- Login requires a verified email (6-digit code). Auth endpoints are rate limited: 10 requests/min per IP.
- Free plan limits: 25 jobs, 1 uploaded resume, 1 builder resume, 1 AI request/month, **0 cover letters**.
- `FEATURES.PAYMENTS=false` in prod: Upgrade buttons/banners/pricing modal render nothing. Settings shows a "Payments are temporarily disabled" notice.
- Onboarding wizard completion is stored per browser (localStorage), not per account.

Statuses: PASS / FAIL / BLOCKED (cannot execute: env, email, plan) / SKIPPED. Severity for FAIL: Blocker, Critical, Major, Minor, Trivial.

Naming: every created entity is prefixed `QA-<block>-` (e.g. `QA-C-Job 01`). Accounts: `qa-regress-<block>@…` as issued by the orchestrator.

---

## Block A — Auth & session (account: A, fresh)

| ID | Title | Steps | Expected |
|----|-------|-------|----------|
| A-01 | Register modal validation | Open `/register`. Submit empty; then email `bad`, password `short`, confirm mismatch. | Field errors: required / "Invalid email address" / "Password must be at least 8 characters" / "Passwords don't match". No request sent. |
| A-02 | Register success → code step | Register with the block account and a valid 8+ char password. | Modal switches to "Enter code" step showing the email. (Verification itself done by orchestrator with user-supplied code; if code is not available mark A-03 BLOCKED.) |
| A-03 | Verify code → auto-login | Enter the 6-digit code. | User is logged in and lands on `/app` → `/app/jobs`. |
| A-04 | Duplicate email register | Register again with the same email. | Error shown on the email field (note: raw backend message expected, record exact text). |
| A-05 | Login wrong password | `/login`, correct email, wrong password. | "Invalid credentials" under password field; stays on modal. |
| A-06 | Login success + redirect guard | Login with valid creds. Then open `/login`, `/register`, `/forgot-password` while logged in. | Lands on `/app/jobs`. Each auth route redirects to `/app`. |
| A-07 | Protected route unauthenticated | In a fresh context open `/app/jobs`, `/app/settings`, `/app/analytics`. | Redirected to `/login` (modal over landing). |
| A-08 | Forgot password flow (UI only) | `/forgot-password`: submit invalid email, then the block email. | Step 1 regex error; then moves to step 2 (code + new password, no confirm field). Resend shows 60 s cooldown. Do not exhaust resends (max 1 send). |
| A-09 | Reset-password page invalid link | Open `/reset-password` with no params, then with `?email=x@y.z&code=123`. | "Invalid link" state with "Send reset code" and "Back to home" in both cases. |
| A-10 | Logout from sidebar and from Settings | Logout via sidebar; login again; logout via Settings → Account → Logout. | Both return to `/`; `/app/*` afterwards redirects to `/login`; localStorage `jobber-auth` cleared. |

## Block B — Onboarding, app shell, theme, language (account: B, fresh, never logged in before)

| ID | Title | Steps | Expected |
|----|-------|-------|----------|
| B-01 | Welcome wizard appears on first login | Login with a fresh account, clean browser profile. | 8-step wizard overlay shown (Welcome … Done), step dots visible, page scroll locked. |
| B-02 | Wizard navigation | Click Next through all steps; Back on step 3; finish with "Let's go". | Steps advance/retreat correctly; sidebar item pulses on steps 1–5; finish navigates to `/app/companies`; wizard does not reappear after reload. |
| B-03 | Wizard exit panel | Clear flags (localStorage `jobber-onboarding-completed`, sessionStorage `jobber-onboarding-dismissed`), reload. Press Escape; choose "Not now"; reload. Then reopen via Settings → Restart tour, choose "Don't show this again". | Escape opens "Leave the tour?" panel. "Not now" hides for session, reappears in a new context. "Don't show again" sets permanent flag. Restart tour from Settings works and navigates to `/app/jobs`. |
| B-04 | Sidebar navigation | Click each of Applications, Resumes, Companies, Cover Letters, Stages, Analytics, user link. | URLs `/app/jobs`, `/app/resumes`, `/app/companies`, `/app/cover-letters`, `/app/stages`, `/app/analytics`, `/app/settings`. Active item highlighted. |
| B-05 | Sidebar collapse persists | Collapse sidebar, reload; expand, reload. | Width 64px / 256px persists via localStorage `jobber-sidebar`; collapsed items show `title` tooltips. |
| B-06 | Theme toggle persists | Toggle Dark in header, reload; set Light in Settings. | `<html class>` switches light/dark; persists via `jobber-theme`; `aria-pressed` correct on Settings buttons. |
| B-07 | Language switch (uk, ru, en) | Switch language in header and in Settings. | Sidebar/labels change language; `<html lang>` updates; persisted in `language`; survives reload. No untranslated keys (e.g. `jobs.board.emptyColumn`) visible on Jobs/Settings pages in each language. |
| B-08 | Legacy routes redirect | Open `/app`, `/app/applications`, `/app/resume-builder`. | → `/app/jobs`, `/app/jobs`, `/app/resumes`. |
| B-09 | Unknown app route | Open `/app/does-not-exist` and `/does-not-exist`. | 404 page "Page not found" with "Back to home" and "Go to dashboard" working. |
| B-10 | Mobile drawer (390x844) | Open `/app/jobs` on mobile viewport; tap hamburger; navigate; press Escape. | Off-canvas drawer opens (`aria-expanded=true`), closes on navigation and on Escape; no horizontal page scroll. |

## Block C — Jobs list: CRUD, search, sort, filter, pagination (account: C)

| ID | Title | Steps | Expected |
|----|-------|-------|----------|
| C-01 | Empty state | Open `/app/jobs` on an account with no jobs. | First-run empty state with "create first job" button that opens Create modal. |
| C-02 | Create modal validation | Open Create; leave title empty; type URL `notaurl`. | Create button disabled with empty title; URL field native validation blocks submit. |
| C-03 | Create job with quick-add company | Title `QA-C-Job 01`, quick-add company `QA-C-Co Alpha` (type + Enter), source `LinkedIn`, description text. Save. | Toast success for company and job; job card appears with company name. |
| C-04 | Create job "already applied" | Create `QA-C-Job 02`, toggle "Already applied", pick applied date. | Card shows applied time-ago; no console errors. |
| C-05 | Create 22 more jobs for pagination | Create `QA-C-Job 03`…`QA-C-Job 24` (API or UI). | 24 jobs total; list shows 20 + pagination "Page 1 of 2". |
| C-06 | Pagination | Next / Previous; open `?page=99`; open `?page=abc`. | Page 2 shows 4 jobs; out-of-range corrects to last page; malformed falls back to page 1. |
| C-07 | Search | Type `Job 02` (debounce), then `zzzz`, then Clear search. | Only matching card; "No results for 'zzzz'" + Clear search restores list; `q` in URL. |
| C-08 | Sort | Switch sort to Title; click Title again; switch to Company. | Order changes asc/desc; URL `sort=` reflects; direction flips on second click. |
| C-09 | Archived filter | From job detail archive `QA-C-Job 24`; back in list use Active / Archived / All. | Archived job hidden in Active, shown in Archived (badge "Archived") and in All; `archived=` in URL. |
| C-10 | Delete job | Three-dot menu → Delete on `QA-C-Job 23`; Cancel; repeat and confirm. | Confirm dialog names the job; Cancel keeps it; Delete removes it with success toast. Then create job #25 and verify Create button becomes **disabled** at limit (record whether any explanation is shown). |

## Block D — Stage templates & Kanban board (account: D)

| ID | Title | Steps | Expected |
|----|-------|-------|----------|
| D-01 | Recommended stages "Add All" | `/app/stages` → Add All. | 7 stages created in order; recommended items show "Added". |
| D-02 | Create custom stage + validation | Create with empty name (disabled), then `QA-D-Take-home`, order 3, kind interview. Try duplicate name. | Submit disabled when empty; stage created with toast; duplicate shows "A stage with this name already exists". |
| D-03 | Edit stage | Edit `QA-D-Take-home` → rename to `QA-D-Take-home v2`, kind other. | Toast; list reflects changes. |
| D-04 | Reorder stages | Move a stage up/down with arrows; reload. | Order persists after reload. |
| D-05 | Board view toggle persists | `/app/jobs` → Board; reload. | Board view stays (localStorage `apps-view-mode`). |
| D-06 | Board columns reflect templates | Create 3 jobs `QA-D-Job 1..3` (one with no stage). Open Board. | Columns in template order; "No Stage" column first and only when such a job exists; empty columns show empty text. |
| D-07 | Drag card to another column (pointer) | Drag `QA-D-Job 1` from current column to "Technical Interview". | Card moves optimistically; toast "Application moved successfully"; persists on reload; job detail shows new stage. |
| D-08 | Keyboard drag | Focus a card, Space, ArrowRight, Space. | Card moves to the next column; toast; persists. |
| D-09 | Card menu actions | Card three-dot → Add comment (text), Add stage; Escape closes menu. | Comment and stage saved (visible in job detail); menu keyboard accessible. |
| D-10 | Delete stage in use / mobile accordion | Try deleting a template used by `QA-D-Job 1`. Then open Board at 390x844. | Error "Cannot delete stage template: it is still used in active applications". Mobile shows accordion of columns, first expanded, no DnD. Record whether delete of an unused template shows any success toast. |

## Block E — Job detail (account: E)

Precondition: agent creates stages (Add All) and job `QA-E-Job Main` with company `QA-E-Co`.

| ID | Title | Steps | Expected |
|----|-------|-------|----------|
| E-01 | Open detail from list, Back keeps filters | Search `Main`, open card, click "Back to jobs". | Detail shows title/company; Back returns to list with `q=Main` preserved. |
| E-02 | Edit fields + Save/Discard | Change title, URL, source, description → Discard; change again → Save. | Save/Discard appear only when dirty; Discard reverts; Save persists after reload. Save disabled with blank title. |
| E-03 | Unsaved changes on navigate away | Edit title, click sidebar "Companies", come back. | Record behavior: expected per spec is no guard (edits lost silently). Mark FAIL only if the app errors; otherwise PASS with Observation "no unsaved-changes guard". |
| E-04 | Move to stage via selector + Complete | Pipeline card: Move to "Phone Screen"; click Complete. | Toast "Application moved successfully"; stage shows completed; timeline updated. |
| E-05 | Add stage via Timeline + change status | Timeline → Add new stage (Technical Interview, comment). Then Change status → skipped. | Toasts; stage appears; status badge updates; submit disabled when status unchanged. |
| E-06 | Delete stage from timeline | Stage menu → Delete stage, confirm. | Stage removed; no console errors. |
| E-07 | Comments CRUD | Add comment, edit it, delete it (confirm names first line). | "edited" marker after edit; toasts for update/delete. |
| E-08 | Reminders | Add reminder with datetime; mark done; delete. | Reminder listed; checkbox toggles done; delete removes; hint text about no emails visible. |
| E-09 | Tags | Create tag `QA-E-Tag` with color (attached automatically); detach; attach existing again. | Tag chip appears/disappears; toasts on create. |
| E-10 | Favorite, Archive/Unarchive, Delete | Toggle heart; Archive (→ list); reopen from Archived filter → Unarchive; Delete with confirm. | Heart toggles; archive toast + navigate to `/app/jobs`; unarchive toast; delete navigates to list and card gone. Also open `/app/jobs/00000000-0000-0000-0000-000000000000` → "Not found" error state with Retry. |

## Block F — Companies & Support widget (account: F)

| ID | Title | Steps | Expected |
|----|-------|-------|----------|
| F-01 | Empty state | `/app/companies` with no companies. | "No companies" with create button. |
| F-02 | Create validation | Open Create; empty name. | Button disabled; location/notes optional. |
| F-03 | Create companies | Create `QA-F-Co Beta` (location, notes) and `QA-F-Co Alpha`. | Toasts; cards show avatar, name, location, notes, "No applications yet", status Idle. |
| F-04 | Edit company | Edit Beta → change location and notes. | Toast; card updated after close. |
| F-05 | Sort | Toggle Name asc/desc, Last activity, Applications count. | Order changes accordingly (Alpha before Beta on name asc). |
| F-06 | Favorite | Click heart on Alpha; reload. | Optimistic toggle persists. |
| F-07 | Stats and "View applications" | Create job `QA-F-Job` for Beta with a stage. Back to companies. Click "View applications". | Card shows 1 total / 1 active, status Active; navigates to `/app/jobs?company_id=…` with "Filtered by company" banner; Clear removes filter. |
| F-08 | Delete company with and without data | Delete Beta (has job): read dialog; cancel. Delete Alpha: confirm. | Beta dialog warns "has related data" with counts; Alpha dialog says safe; Alpha removed. Record whether a success toast appears (spec: none). |
| F-09 | Support widget validation | Floating Support button: subject 2 chars, message 5 chars. | Send disabled; live "N more char(s)" counter; limits 200/2000. |
| F-10 | Support widget send | Subject `QA-F regression test, please ignore`, message ≥10 chars. Send once. | Success toast "Your message has been sent…" (or, if `support/status` says unavailable, button hidden → BLOCKED with note). |

## Block G — Resumes: upload, list, limits (account: G)

| ID | Title | Steps | Expected |
|----|-------|-------|----------|
| G-01 | Empty state + Create dropdown | `/app/resumes`. | "No resumes" empty state; Create dropdown has Upload / Build / Import, all enabled. |
| G-02 | Upload rejects non-PDF and fake PDF | Upload `.txt`; upload a `.txt` renamed to `.pdf`. | "Only PDF allowed"; "not a real PDF" error. |
| G-03 | Upload rejects >10 MB | Generate 11 MB PDF, upload. | File size error, no upload request. |
| G-04 | Upload valid PDF | Upload a small valid PDF (title left empty). | Title auto-fills from filename; progress shown; success toast; card with "Uploaded" badge, Active, "Not used yet". |
| G-05 | Download signed link | Click Download on uploaded card. | New tab/URL to S3 signed link (or download started); no error toast. |
| G-06 | Upload limit (Free = 1) | Open Create dropdown again. | "Upload resume" disabled. Record whether any explanation/banner is shown (spec: none while payments off). |
| G-07 | Create resume by external URL | Create → URL method, title `QA-G-URL resume`, URL `https://example.com/cv.pdf`. | Card with "View resume" link opening the URL. (If URL method counts toward the limit and is disabled, mark BLOCKED with evidence.) |
| G-08 | Edit uploaded resume | Edit → rename `QA-G-Uploaded v2`, toggle Active off. | Toast; badge Inactive; title updated. |
| G-09 | Delete protection when used | Create job `QA-G-Job` with "Already applied" and select the uploaded resume. Back to resumes. | Card shows "Used in 1 applications"; Delete disabled with tooltip "Cannot delete resume used in applications". |
| G-10 | Filters & sort | Build a resume (Create → Build, then Back). Use tabs All/Uploaded/Built; sort Title/Created. | Tabs filter correctly; "No resumes for filter" when applicable; sort order changes. Then Build/Import disabled at builder limit (Free = 1). |

## Block H — Resume Builder editor (account: H)

Precondition: Create → Build resume (lands in editor).

| ID | Title | Steps | Expected |
|----|-------|-------|----------|
| H-01 | Editor loads, toolbar, 404 | Editor shows toolbar (Back, title, Undo/Redo disabled, Export, panel toggles). Open `/app/resume-builder/00000000-0000-0000-0000-000000000000`. | Toolbar correct; invalid id shows "Resume not found" + Back to list. |
| H-02 | Inline edit + autosave | Click name field in preview, type `QA-H Tester`, Enter. Wait 2 s. Reload. | Indicator Unsaved → Saving… → Saved; value persists after reload. |
| H-03 | Escape cancels inline edit | Click summary, type text, press Escape. | Edit cancelled, original value retained. |
| H-04 | Undo / Redo buttons and shortcuts | Make 3 edits; click Undo twice; Redo once; blur inputs and press Ctrl+Z. | State reverts/reapplies correctly; buttons enable/disable; shortcut works outside inputs. |
| H-05 | Template switch | Design → Template → pick Modern, then Timeline. | Preview re-renders; selection persists after reload. |
| H-06 | Colors & typography | Pick a palette; custom color; font Arial; size 12; spacing 120%. | Preview updates; values persist after reload. |
| H-07 | Layout & sections | Rename title to `QA-H Resume`; set margins; switch to double-left; hide Skills; re-add via "Add section"; move Education above Experience. | Preview reflects every change; order persists. |
| H-08 | Export PDF and DOCX | Export → PDF; Export → DOCX. | Downloads `QA-H Resume.pdf` / `.docx` (non-empty, PDF starts with `%PDF`); success toasts. |
| H-09 | ATS checker | Open ATS panel → Check. | Score circle + issues/keywords, or inline monthly-limit message with reset date if AI limit already used (then PASS with note). No blank/unhandled state. |
| H-10 | Content library + import by text | Library: add snippet `QA-H snippet`, insert, edit, delete. Back to `/app/resumes` → Import → Paste text. | Snippet CRUD toasts; insert adds text. Import disabled at builder limit (Free = 1): record behavior (spec: disabled, no banner). Also check `beforeunload` prompt fires with unsaved edits. |

## Block I — Cover letters, Analytics, Sharing (account: I)

Precondition (after I-06): Add All stages; create 6 jobs `QA-I-Job 1..6` across stages (2 Applied, 2 Phone Screen, 1 Offer, 1 Rejected), with 2 different sources and 1 with a resume.

| ID | Title | Steps | Expected |
|----|-------|-------|----------|
| I-01 | Cover letters on Free plan | `/app/cover-letters`. | Empty state; Create disabled (Free = 0). Record whether any explanation is shown. If Create is enabled, create `QA-I-Letter` and run I-02; else I-02 BLOCKED (plan). |
| I-02 | Cover letter editor smoke | Edit title, pick template, change font/size/color, add paragraph, export PDF. | Autosave indicator; template picker (10 templates); exports download. |
| I-03 | Analytics overview cards | `/app/analytics`. | Total 6, active/closed consistent with stages, response rate and avg response time numeric (no NaN/undefined). |
| I-04 | Funnel | Inspect funnel section. | Converted / drop-off / rejected per stage match created data; stage breakdown lists stages by kind. |
| I-05 | Time in stage, Resume effectiveness, Sources | Scroll remaining tables. | Each renders rows or its own empty state; sources table shows the 2 sources with correct counts; bar chart renders. |
| I-06 | Analytics empty state | Run FIRST, before creating any data: open `/app/analytics` on the fresh account. | Overall "No data" empty state, no NaN/undefined, no console errors. |
| I-07 | Create share link | Analytics → Share → Create share link. | Link created, clipboard toast ("Link copied" or "Failed to copy" in headless — both acceptable), listed with Revoke. |
| I-08 | Public shared stats page | Open `/s/<token>` in a fresh (logged-out) context. | "Job Search Stats", "Shared on <date>", overview + funnel; CTA "Get started free" → `/register`; `noindex` meta present. |
| I-09 | Revoke share link | Revoke; reload `/s/<token>`. | Toast "Share link revoked"; page shows "Link not found". Also `/s/invalidtoken` → "Link not found". |
| I-10 | Share limit | Create links until `SHARE_LIMIT_REACHED` (stop at 10 attempts). | Error "Share link limit reached…" appears at the limit; record the numeric limit. Revoke all afterwards. |

## Block J — Public pages, legal, blog, consent, settings, responsive (account: J, only for settings cases)

| ID | Title | Steps | Expected |
|----|-------|-------|----------|
| J-01 | Landing renders, no console errors | Open `/` (desktop). | Hero, Features, How it works, Pricing (Free $0, Pro $7, Enterprise $19), FAQ, Footer. Record all console errors / CSP violations and failed requests. |
| J-02 | Navbar anchors & Features dropdown | Click How it works, Pricing, FAQ; open Features dropdown → each item. | Smooth scroll to sections; dropdown links to `/features/applications`, `/features/resume-builder`, `/features/cover-letters`. `/features` → `/features/applications`. |
| J-03 | Feature pages CTA | On each `/features/*` page click primary CTA (logged out). | → `/register` modal. |
| J-04 | Cookie consent | Fresh context: banner visible with Privacy link; click Essential only; reload. Footer → Cookie settings. | Banner hidden after choice (localStorage `cookie-consent`); Cookie settings brings banner back. |
| J-05 | Blog list & post, per language | `/blog` in en, ru, uk; open first post; open `/blog/non-existent`. | List counts: en 8, ru 6, uk 13 (record actual); post renders markdown; unknown slug shows not-found with link to `/blog`. |
| J-06 | Legal pages | `/privacy`, `/terms`, `/refund`. | Render with last-updated dates, mailto link, back link to `/`. |
| J-07 | 404 and www/host checks | Open `/nope`; `https://www.jobber-app.com/`. | 404 page works. Record HTTP status of www host (pre-check showed 525). |
| J-08 | Landing mobile (390x844) | Open `/`; hamburger menu; Escape; `/register` modal on mobile. | Menu opens/closes; no horizontal scroll; auth modal full-screen under 640px. |
| J-09 | Settings page content | Login as J. `/app/settings`. | Profile name/email read-only; Theme & Language controls; Subscription shows Free, payments-disabled notice, usage grid (jobs 0/25, resumes 0/1, AI 0/1, builders 0/1, cover letters "Not included"); Restart tour; Logout. No Upgrade/Calendar. |
| J-10 | SEO & meta | Check `<title>`/description on `/`, `/blog`, `/app/jobs` (logged in). | Public pages have titles + hreflang; app pages have `noindex`. |

---

## Report format (per block)

Each agent writes `reports/<BLOCK>.md`:

```
# Block <X>: <name>
Tester account: …   Run (UTC): start – end
Summary: PASS n / FAIL n / BLOCKED n / SKIPPED n

| ID | Title | Status | Severity | Evidence | Notes |

## Failures   (ID, severity, steps, expected, actual, evidence, reproducibility)
## Blocked    (ID: reason)
## Observations (non-case findings: console errors, UX, typos, slow pages)
## Created test data (accounts, entities)
```
