-- Roll back application first; these tables contain only this feature's history.
DROP TABLE interview_operations;
DROP TABLE interview_receipts;
DROP TABLE interview_answers;
DROP TABLE interview_sessions;
DROP TABLE interview_versions;

DROP TABLE ai_quota_reservations;
