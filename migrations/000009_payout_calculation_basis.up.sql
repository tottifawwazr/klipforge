ALTER TABLE payouts ADD COLUMN basis_views bigint NOT NULL DEFAULT 0 CHECK(basis_views>=0), ADD COLUMN basis_snapshot_at timestamptz;
