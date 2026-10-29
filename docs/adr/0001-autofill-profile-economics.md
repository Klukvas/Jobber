# Autofill Profile extraction uses credits; cached use is free

Updated 2026-10-25 by the Free / Pro credit policy in
[the system specification](../../SYSTEM_SPECIFICATION.md#i8-free--pro-credits-and-bounded-ai-spending-2026-10-25).

Free and Pro may explicitly start an extraction for 7 credits, subject to the
per-user and global dollar guards. The extension confirms the price before
`POST /api/v1/resumes/:id/autofill-profile`. A cache hit is free. The old paid-plan
gate and AI-generating GET are superseded.

