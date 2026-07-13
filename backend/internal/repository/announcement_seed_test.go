package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestEnsureInitialAnnouncementsNilDB(t *testing.T) {
	err := ensureInitialAnnouncements(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil sql db")
}

func TestEnsureInitialAnnouncementsHasNoHistoricalDefaults(t *testing.T) {
	require.Empty(t, initialAnnouncementSeeds)
}

func TestUpsertInitialAnnouncements(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	seed := initialAnnouncementSeed{
		SeedKey:    "release_notice_2026_07_13",
		Title:      "Release notice",
		Content:    "Release content",
		NotifyMode: "silent",
		Published:  time.Date(2026, 7, 13, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60)),
	}
	mock.ExpectExec(regexp.QuoteMeta(initialAnnouncementUpsertSQL)).
		WithArgs(seed.SeedKey, seed.Title, seed.Content, seed.NotifyMode, seed.Published).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, upsertInitialAnnouncements(context.Background(), db, []initialAnnouncementSeed{seed}))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertInitialAnnouncementsReturnsExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	wantErr := errors.New("insert failed")
	seed := initialAnnouncementSeed{
		SeedKey:    "failed_notice",
		Title:      "Failed notice",
		Content:    "Failed content",
		NotifyMode: "silent",
		Published:  time.Now(),
	}
	mock.ExpectExec(regexp.QuoteMeta(initialAnnouncementUpsertSQL)).
		WithArgs(seed.SeedKey, seed.Title, seed.Content, seed.NotifyMode, seed.Published).
		WillReturnError(wantErr)

	err = upsertInitialAnnouncements(context.Background(), db, []initialAnnouncementSeed{seed})
	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	require.Contains(t, err.Error(), seed.SeedKey)
	require.NoError(t, mock.ExpectationsWereMet())
}
