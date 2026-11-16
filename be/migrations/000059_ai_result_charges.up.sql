BEGIN;
-- Provider states remain an immutable record of liability, including failed calls.
-- Existing dispatched/settled rows are deliberately not presumed successful.
ALTER TABLE ai_credit_operations ADD COLUMN charge_state TEXT NOT NULL DEFAULT 'reserved'
 CHECK (charge_state IN ('reserved','charged','released','legacy'));
ALTER TABLE ai_credit_operations ADD COLUMN charge_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE ai_credit_operations ADD COLUMN charge_finished_at TIMESTAMPTZ;
UPDATE ai_credit_operations SET charge_state=CASE WHEN state='released' THEN 'released' ELSE 'legacy' END;

-- Durable receipts, unlike provider settlement, prove application lifecycle.
UPDATE ai_credit_operations c SET charge_state='reserved'
 WHERE EXISTS(SELECT 1 FROM interview_operations o WHERE o.user_id=c.user_id AND o.id::text=c.operation_key AND o.status IN ('queued','running'))
 OR EXISTS(SELECT 1 FROM resume_tailoring_operations o WHERE o.user_id=c.user_id AND o.ai_receipt_id::text=c.operation_key AND o.status IN ('pending','parsed'));
UPDATE ai_credit_operations c SET charge_state='charged',charge_reason='historical_saved_result',charge_finished_at=now()
 WHERE EXISTS(SELECT 1 FROM interview_operations o WHERE o.user_id=c.user_id AND o.id::text=c.operation_key AND o.status='succeeded')
 OR EXISTS(SELECT 1 FROM resume_tailoring_operations o WHERE o.user_id=c.user_id AND o.ai_receipt_id::text=c.operation_key AND o.status='succeeded');

-- Refund only outcomes proved by a durable receipt, never arbitrary old settled rows.
UPDATE ai_credit_operations c SET charge_state='released',charge_reason='historical_failed_result',charge_finished_at=now()
 WHERE EXISTS (SELECT 1 FROM interview_operations o WHERE o.user_id=c.user_id AND o.id::text=c.operation_key AND o.status IN ('failed','interrupted'))
 OR EXISTS (SELECT 1 FROM resume_tailoring_operations o WHERE o.user_id=c.user_id AND o.ai_receipt_id::text=c.operation_key AND o.status IN ('failed','interrupted'));
UPDATE interview_operations o SET charge_released=true WHERE status IN ('failed','interrupted')
 AND EXISTS(SELECT 1 FROM ai_credit_operations c WHERE c.user_id=o.user_id AND c.operation_key=o.id::text AND c.charge_state='released');
CREATE INDEX ai_credit_operations_charge_grant ON ai_credit_operations(grant_id) WHERE charge_state<>'released';
COMMIT;
