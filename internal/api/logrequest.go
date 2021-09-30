package api

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"sort"

	"github.com/sirupsen/logrus"
)

// logRequest logs the HTTP request details.
// This is helpful for debugging.
func logRequest(ctx context.Context, r *http.Request, logBody bool) {
	logrus.WithContext(ctx).Infof("Protocol: %s", r.Proto)
	logrus.WithContext(ctx).Infof("Host: %s", r.Host)
	logrus.WithContext(ctx).Infof("Method: %s", r.Method)
	logrus.WithContext(ctx).Infof("Request: %s", r.RequestURI)

	headerNames := make([]string, 0, len(r.Header))
	for headerName := range r.Header {
		headerNames = append(headerNames, headerName)
	}
	sort.Strings(headerNames)
	logrus.WithContext(ctx).Infof("Headers: (%d)", len(r.Header))
	for _, headerName := range headerNames {
		for _, value := range r.Header[headerName] {
			logrus.WithContext(ctx).Infof("* %s: %s", headerName, value)
		}
	}

	if logBody {
		// Read the body.
		contents, _ := ioutil.ReadAll(r.Body)
		// Put the body back the way that we found it.
		r.Body = ioutil.NopCloser(bytes.NewReader(contents))

		logrus.WithContext(ctx).Infof("Body: (%d)", len(contents))
		if len(contents) > 500 {
			logrus.WithContext(ctx).Infof("%s[truncated]", string(contents[:500]))
		} else {
			logrus.WithContext(ctx).Infof("%s", string(contents))
		}
	}
}
