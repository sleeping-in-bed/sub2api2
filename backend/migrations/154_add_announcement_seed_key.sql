ALTER TABLE announcements ADD COLUMN IF NOT EXISTS seed_key VARCHAR(120);

CREATE UNIQUE INDEX IF NOT EXISTS idx_announcements_seed_key_unique
    ON announcements(seed_key)
    WHERE seed_key IS NOT NULL;

COMMENT ON COLUMN announcements.seed_key IS '种子公告幂等键';
