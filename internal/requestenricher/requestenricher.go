package requestenricher

import (
	"bytes"
	"context"
	"io"
	"net/http"

	re "form3interview/pkg/requestenricher"
)

type EnrichedHttpClient struct {
	client http.Client
}

func EnrichClient(client http.Client) EnrichedHttpClient {
	return EnrichedHttpClient{client: client}
}

func (c EnrichedHttpClient) Do(req *http.Request, enricher ...re.RequestEnricher) (*http.Response, error) {
	req = req.WithContext(c.getCtx(enricher...))

	c.getBeforeHook(enricher...)()
	resp, err := c.client.Do(req)
	if err != nil {
		return resp, err
	}

	enResp := cloneResponse(resp)
	c.getAfterHook(enricher...)(enResp)
	return resp, err
}

func (c EnrichedHttpClient) getCtx(en ...re.RequestEnricher) context.Context {
	if len(en) == 0 || en[0].Ctx == nil {
		return context.TODO()
	}

	return en[0].Ctx
}

func (c EnrichedHttpClient) getBeforeHook(en ...re.RequestEnricher) func() {
	if len(en) == 0 || en[0].BeforeHook == nil {
		return func() {}
	}

	return en[0].BeforeHook
}

func (c EnrichedHttpClient) getAfterHook(en ...re.RequestEnricher) func(*http.Response) {
	if len(en) == 0 || en[0].AfterHook == nil {
		return func(*http.Response) {}
	}

	return en[0].AfterHook
}

func cloneResponse(resp *http.Response) *http.Response {
	return &http.Response{
		Status:           resp.Status,
		StatusCode:       resp.StatusCode,
		Proto:            resp.Proto,
		ProtoMajor:       resp.ProtoMajor,
		ProtoMinor:       resp.ProtoMinor,
		Header:           resp.Header,
		Body:             io.NopCloser(bytes.NewReader([]byte{})),
		ContentLength:    resp.ContentLength,
		TransferEncoding: resp.TransferEncoding,
		Close:            resp.Close,
		Uncompressed:     resp.Uncompressed,
		Trailer:          resp.Trailer,
		Request:          resp.Request,
		TLS:              resp.TLS,
	}
}
