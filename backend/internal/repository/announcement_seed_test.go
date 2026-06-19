package repository

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestEnsureInitialAnnouncementsNilDB(t *testing.T) {
	err := ensureInitialAnnouncements(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nil sql db")
}

func TestEnsureInitialAnnouncementsUpsertsAllSeeds(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	for _, seed := range initialAnnouncementSeeds {
		mock.ExpectExec(regexp.QuoteMeta(initialAnnouncementUpsertSQL)).
			WithArgs(seed.SeedKey, seed.Title, seed.Content, seed.NotifyMode, seed.Published).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	require.NoError(t, ensureInitialAnnouncements(context.Background(), db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEnsureInitialAnnouncementsReturnsExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	wantErr := errors.New("insert failed")
	seed := initialAnnouncementSeeds[0]
	mock.ExpectExec(regexp.QuoteMeta(initialAnnouncementUpsertSQL)).
		WithArgs(seed.SeedKey, seed.Title, seed.Content, seed.NotifyMode, seed.Published).
		WillReturnError(wantErr)

	err = ensureInitialAnnouncements(context.Background(), db)
	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	require.Contains(t, err.Error(), seed.SeedKey)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInitialAnnouncementSeedsAreStableAndSanitized(t *testing.T) {
	seen := make(map[string]struct{}, len(initialAnnouncementSeeds))
	for _, seed := range initialAnnouncementSeeds {
		require.NotEmpty(t, seed.SeedKey)
		_, exists := seen[seed.SeedKey]
		require.False(t, exists, "duplicate announcement seed key %s", seed.SeedKey)
		seen[seed.SeedKey] = struct{}{}

		body := seed.Title + "\n" + seed.Content
		for _, prohibited := range []string{"售后群", "闲鱼", "咸鱼"} {
			require.False(t, strings.Contains(body, prohibited), "announcement %s contains %s", seed.SeedKey, prohibited)
		}
	}
}
