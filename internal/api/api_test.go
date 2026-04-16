package api_test

import (
	"net/http"
	"testing"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/applicationtest"
	"github.com/downballot/downballot/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tekkamanendless/httperror"
)

func TestAPI(t *testing.T) {
	testutils.Setup(t)

	ctx := t.Context()

	application := applicationtest.New(t, ctx)
	t.Cleanup(func() {
		application.Close()
	})

	t.Run("Bogus", func(t *testing.T) {
		err := application.UnauthenticatedClient().Do(ctx, http.MethodGet, "/api/bogus", nil, nil)
		require.ErrorIs(t, err, httperror.ErrStatusNotFound)
	})
	t.Run("Health", func(t *testing.T) {
		var output downballotapi.HealthCheckResponse
		err := application.UnauthenticatedClient().Do(ctx, http.MethodGet, "/api/v1/health/check", nil, &output)
		require.NoError(t, err)
		assert.True(t, output.Healthy)
	})
}
