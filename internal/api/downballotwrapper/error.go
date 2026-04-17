package downballotwrapper

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/downballot/downballot/downballotapi"
	"github.com/emicklei/go-restful/v3"
	"github.com/tekkamanendless/httperror"
	"github.com/threatmate/restfulwrapper"
)

// Error returns a wrapped error that will be rendered to JSON using the envelope.
func Error(err error) error {
	return &wrappedError{
		err: err,
	}
}

// wrappedError is an error
type wrappedError struct {
	err error
}

var _ error = (*wrappedError)(nil)
var _ restfulwrapper.ErrorWriter = (*wrappedError)(nil)

func (e *wrappedError) Error() string {
	return e.err.Error()
}

func (e *wrappedError) WriteError(resp *restful.Response) {
	type Output struct {
		Type string `json:"type"`
	}

	code := http.StatusInternalServerError
	{
		var errStatus *httperror.Error
		if errors.As(e.err, &errStatus) {
			code = errStatus.Code()
		}
	}

	content := downballotapi.Envelope[Output]{
		Message: e.err.Error(),
		Success: false,
		Data: Output{
			Type: fmt.Sprintf("%T", e.err),
		},
	}
	resp.WriteHeaderAndEntity(code, content)
}
