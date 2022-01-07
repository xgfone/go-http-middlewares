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
	middlewares "github.com/xgfone/go-http-middlewares"
)

// Middleware returns a new http handler middleware supporting prometheus.
func Middleware(sh *ServerHandler) middlewares.Middleware {
	sh.Init()
	return func(wrappedNext http.Handler) (new http.Handler) {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			sh.Serve(wrappedNext, rw, r)
		})
	}
}

// ServerHandler is a http handler that is used to trace the http request
// by opentracing, which extracts the span context from the http request
// header, creates a new span as the server span from the span context,
// and put it into the request context.
type ServerHandler struct {
	http.Handler
	Option

	// Start is called before starting the tracer if it is set,
	// which may be used to wrap the response writer.
	//
	// Notice: the tracer is not enabled, it will be ignored.
	Start func(http.ResponseWriter, *http.Request) (http.ResponseWriter, *http.Request)

	// End is called after the request ends if it is set,
	// which may be used to set the status code tag, for example,
	//
	//   ext.HTTPStatusCode.Set(sp, rw.(StatusCodeInterface).StatusCode())
	//
	// Notice: the tracer is not enabled, it will be ignored.
	End func(http.ResponseWriter, *http.Request, opentracing.Span)
}

// NewServerHandler returns a new ServerHandler to trace the request.
func NewServerHandler(handler http.Handler) *ServerHandler {
	sh := &ServerHandler{Handler: handler}
	sh.Option.Init()
	return sh
}

// WrappedHandler returns the wrapped http handler.
func (sh *ServerHandler) WrappedHandler() http.Handler { return sh.Handler }

// Serve uses the given http handler to serve the request with w and r.
func (sh *ServerHandler) Serve(h http.Handler, w http.ResponseWriter, r *http.Request) {
	if sh.SpanFilter(r) {
		sh.Handler.ServeHTTP(w, r)
		return
	}

	if sh.Start != nil {
		w, r = sh.Start(w, r)
	}

	tracer := sh.GetTracer()
	sc, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
	sp := tracer.StartSpan(sh.OperationNameFunc(r), ext.RPCServerOption(sc))
	ext.HTTPMethod.Set(sp, r.Method)
	ext.Component.Set(sp, sh.GetComponentName(r))
	ext.HTTPUrl.Set(sp, sh.URLTagFunc(r.URL))
	sh.SpanObserver(r, sp)

	r = r.WithContext(opentracing.ContextWithSpan(r.Context(), sp))
	defer sp.Finish()
	if sh.End != nil {
		defer sh.End(w, r, sp)
	}

	sh.Handler.ServeHTTP(w, r)
}

// HandleHTTP implements the interface middlewares.Handler, which is equal to
//
//   return sh.Serve(sh.Handler, rw, req)
//
func (sh *ServerHandler) HandleHTTP(rw http.ResponseWriter, req *http.Request) error {
	return sh.Serve(sh.Handler, rw, req)
}

// ServeHTTP implements the interface http.Handler, which is equal to
//
//   sh.Serve(sh.Handler, rw, req)
//
func (sh *ServerHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	sh.Serve(sh.Handler, rw, req)
}
