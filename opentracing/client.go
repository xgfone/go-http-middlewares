// Copyright 2021 xgfone
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package opentracing

import (
	"net/http"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// RoundTripper is a RoundTripper to support OpenTracing,
// which extracts the parent span from the context of the sent http.Request,
// then creates a new span by the context of the parent span for http.Request.
type RoundTripper struct {
	http.RoundTripper
	Option
}

// NewRoundTripper returns a new RoundTripper.
//
// If rt is nil, use http.DefaultTransport instead.
func NewRoundTripper(rt http.RoundTripper) *RoundTripper {
	roundTripper := &RoundTripper{RoundTripper: rt}
	roundTripper.Option.Init()
	return roundTripper
}

// WrappedRoundTripper returns the wrapped http.RoundTripper.
func (rt *RoundTripper) WrappedRoundTripper() http.RoundTripper {
	return rt.RoundTripper
}

func (rt *RoundTripper) roundTrip(req *http.Request) (*http.Response, error) {
	if rt.RoundTripper == nil {
		return http.DefaultTransport.RoundTrip(req)
	}
	return rt.RoundTripper.RoundTrip(req)
}

// RoundTrip implements the interface http.RounderTripper.
func (rt *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.SpanFilter(req) {
		return rt.roundTrip(req)
	}

	ctx := req.Context()
	opts := []opentracing.StartSpanOption{ext.SpanKindRPCClient}
	if pspan := opentracing.SpanFromContext(ctx); pspan != nil {
		opts = []opentracing.StartSpanOption{opentracing.ChildOf(pspan.Context())}
	}

	tracer := rt.GetTracer()
	sp := tracer.StartSpan(rt.OperationNameFunc(req), opts...)
	ext.HTTPUrl.Set(sp, rt.URLTagFunc(req.URL))
	ext.Component.Set(sp, rt.GetComponentName(req))
	ext.HTTPMethod.Set(sp, req.Method)
	rt.SpanObserver(req, sp)
	defer sp.Finish()

	carrier := opentracing.HTTPHeadersCarrier(req.Header)
	tracer.Inject(sp.Context(), opentracing.HTTPHeaders, carrier)

	return rt.roundTrip(req.WithContext(opentracing.ContextWithSpan(ctx, sp)))
}
