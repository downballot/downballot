package api

import (
	"net/http"

	"github.com/downballot/downballot/downballotapi"
	restful "github.com/emicklei/go-restful/v3"
)

func (i *Instance) registerHealthEndpoints(ws *restful.WebService) {
	ws.Route(
		ws.GET("health/check").To(i.healthCheck).
			Doc(`Health check`).
			Notes(`This performs a health check.`).
			Returns(http.StatusOK, "OK", downballotapi.HealthCheckResponse{}),
	)
}

func (i *Instance) healthCheck(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()

	databaseErr := i.App.DB().Exec("SELECT 1").Error

	output := downballotapi.HealthCheckResponse{
		Healthy: databaseErr == nil,
	}

	WriteEntity(ctx, response, output)
}
