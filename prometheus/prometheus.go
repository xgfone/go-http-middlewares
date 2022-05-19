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

package prometheus

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	middlewares "github.com/xgfone/go-http-middlewares"
)

// DefaultHistogramBuckets is the default Histogram buckets.
var DefaultHistogramBuckets = []float64{
	.005, .01, .025, .05, .075,
	.1, .25, .5, .75, 1,
	1.5, 2,
}

// Middleware returns a new http handler middleware supporting prometheus.
func Middleware(option *Option) middlewares.Middleware {
	sh := NewServerHandler(nil)
	if option != nil {
		sh.Option = *option
	}

	sh._init()
	return func(wrappedNext http.Handler) (new http.Handler) {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			sh.Serve(wrappedNext, rw, r)
		})
	}
}

// ResponseWriter is an extended http response writer with the status code.
type ResponseWriter interface {
	http.ResponseWriter
	StatusCode() int
}

// Option is used to configure the prometheus http handler.
type Option struct {
	prometheus.Registerer

	// Default: ""
	Namespace string
	Subsystem string

	// Default: DefaultHistogramBuckets
	Buckets []float64

	// Enable the corresponding label
	Path   bool // Default: false
	Code   bool // Default: true
	Method bool // Default: true
}

// ServerHandler is a http handler to handle the prometheus metrics
// based on RED without E.
type ServerHandler struct {
	http.Handler
	Option

	once             sync.Once
	labels           []string
	requestsTotal    *prometheus.CounterVec
	requestDurations *prometheus.HistogramVec
}

// NewServerHandler returns a new http handler supporting the prometheus metrics
// based on RED.
func NewServerHandler(handler http.Handler) *ServerHandler {
	return &ServerHandler{
		Handler: handler,
		Option:  Option{Method: true, Code: true},
	}
}

// WrappedHandler returns the wrapped http handler.
func (sh *ServerHandler) WrappedHandler() http.Handler { return sh.Handler }

// Init initializes the metrics.
func (sh *ServerHandler) _init() { sh.once.Do(sh._init2) }
func (sh *ServerHandler) _init2() {
	_buckets := sh.Buckets
	if len(_buckets) == 0 {
		_buckets = DefaultHistogramBuckets
	}

	buckets := make([]float64, len(_buckets))
	copy(buckets, _buckets)

	sh.labels = make([]string, 0, 3)
	if sh.Method {
		sh.labels = append(sh.labels, "method")
	}
	if sh.Path {
		sh.labels = append(sh.labels, "path")
	}
	if sh.Code {
		sh.labels = append(sh.labels, "code")
	}

	var factory promauto.Factory
	if sh.Registerer == nil {
		factory = promauto.With(prometheus.DefaultRegisterer)
	} else {
		factory = promauto.With(sh.Registerer)
	}

	sh.requestsTotal = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: sh.Namespace,
		Subsystem: sh.Subsystem,

		Name: "http_requests_total",
		Help: "The total number of the http requests",
	}, sh.labels)

	sh.requestDurations = factory.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: sh.Namespace,
		Subsystem: sh.Subsystem,

		Name: "http_request_duration_seconds",
		Help: "The duration to handle the http request",

		Buckets: buckets,
	}, sh.labels)
}

func (sh *ServerHandler) handle(w http.ResponseWriter, r *http.Request, start time.Time) {
	code := 200
	if rw, ok := w.(ResponseWriter); ok {
		code = rw.StatusCode()
	}

	labels := make(prometheus.Labels, len(sh.labels))
	for _len := len(sh.labels) - 1; _len >= 0; _len-- {
		switch sh.labels[_len] {
		case "code":
			labels["code"] = fmt.Sprint(code)

		case "path":
			labels["path"] = r.URL.Path

		case "method":
			labels["method"] = r.Method
		}
	}

	sh.requestsTotal.With(labels).Inc()
	sh.requestDurations.With(labels).Observe(time.Since(start).Seconds())
}

// Serve uses the given http handler to serve the request with w and r.
func (sh *ServerHandler) Serve(h http.Handler, w http.ResponseWriter, r *http.Request) (err error) {
	sh._init()
	defer sh.handle(w, r, time.Now())
	if hh, ok := h.(middlewares.Handler); ok {
		err = hh.HandleHTTP(w, r)
	} else {
		h.ServeHTTP(w, r)
	}
	return
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
