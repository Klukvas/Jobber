ALTER TABLE jobs ADD COLUMN board_column VARCHAR(20) NOT NULL DEFAULT 'wishlist'
  CHECK (board_column IN ('wishlist', 'applied', 'interview', 'offer', 'rejected'));

CREATE INDEX idx_jobs_board_column ON jobs(user_id, board_column);
