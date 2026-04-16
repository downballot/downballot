package databasetest_test

import (
	"testing"

	"github.com/downballot/downballot/internal/databasetest"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	ctx := t.Context()

	db, err := databasetest.New(ctx)
	require.NoError(t, err)
	require.NotNil(t, db)
}
