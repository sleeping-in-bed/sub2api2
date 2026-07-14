package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const initialAnnouncementUpsertSQL = `
INSERT INTO announcements (
	seed_key,
	title,
	content,
	status,
	notify_mode,
	targeting,
	starts_at,
	ends_at,
	created_at,
	updated_at
)
VALUES (
	$1,
	$2,
	$3,
	'active',
	$4,
	'{}'::jsonb,
	$5,
	NULL,
	$5,
	$5
)
ON CONFLICT (seed_key) WHERE seed_key IS NOT NULL
DO UPDATE SET
	title = EXCLUDED.title,
	content = EXCLUDED.content,
	status = EXCLUDED.status,
	notify_mode = EXCLUDED.notify_mode,
	targeting = EXCLUDED.targeting,
	starts_at = EXCLUDED.starts_at,
	ends_at = EXCLUDED.ends_at,
	updated_at = EXCLUDED.updated_at;
`

type initialAnnouncementSeed struct {
	SeedKey    string
	Title      string
	Content    string
	NotifyMode string
	Published  time.Time
}

var initialAnnouncementSeeds = []initialAnnouncementSeed{}

func ensureInitialAnnouncements(ctx context.Context, db *sql.DB) error {
	return upsertInitialAnnouncements(ctx, db, initialAnnouncementSeeds)
}

func upsertInitialAnnouncements(ctx context.Context, db *sql.DB, seeds []initialAnnouncementSeed) error {
	if db == nil {
		return fmt.Errorf("nil sql db")
	}

	for _, seed := range seeds {
		if _, err := db.ExecContext(
			ctx,
			initialAnnouncementUpsertSQL,
			seed.SeedKey,
			seed.Title,
			seed.Content,
			seed.NotifyMode,
			seed.Published,
		); err != nil {
			return fmt.Errorf("seed initial announcement %s: %w", seed.SeedKey, err)
		}
	}

	return nil
}
