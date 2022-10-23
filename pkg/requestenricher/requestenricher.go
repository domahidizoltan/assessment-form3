package requestenricher

import (
	"context"
	"net/http"
)

type RequestEnricher struct {
	Ctx        context.Context
	BeforeHook func()
	AfterHook  func(*http.Response)
}
