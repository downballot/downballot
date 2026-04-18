package api

import (
	"context"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/threatmate/restfulwrapper"
)

type GetHealthCheckMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.UseDatabase
	_ string `api:"httppath:/health/check"`
	_ string `api:"doc" description:"Perform a health check."`
	_ string `api:"notes" description:"This performs a health check"`
}

func (a *API) GetHealthCheck(ctx context.Context, meta GetHealthCheckMetadata) (output downballotapi.Envelope[downballotapi.HealthCheckResponse], err error) {
	databaseErr := meta.DB.Exec("SELECT 1").Error

	output.Message = "OK"
	output.Success = true
	output.Data.Healthy = databaseErr == nil

	return output, nil
}
