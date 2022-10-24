// Package requestenricher provides utilities to give the client's caller more control over the requests.
package requestenricher

import (
	"context"
	"net/http"
)

// RequestEnricher is passed to every client request and it helps the caller to have more control over the requests.
// This could be helpful on using custom context or instrumenting the client calls i.e. for measuring request time.
type RequestEnricher struct {
	// Ctx is used to pass the callers context which may have a timeout for instance.
	Ctx context.Context
	// BeforeHook is a function which runs before the client request.
	BeforeHook func()
	// AfterHook is a function which runs after the client request.
	// The http response is passed without the body so the caller can inspect headers and other details.
	AfterHook func(*http.Response)
}
