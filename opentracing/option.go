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
	"fmt"
	"net/http"
	"net/url"

	"github.com/opentracing/opentracing-go"
)

// Option is used to configure the OpenTracing middleware and RoundTripper.
type Option struct {
	Tracer        opentracing.Tracer // Default: opentracing.GlobalTracer()
	ComponentName string             // Default: use ComponentNameFunc(req)

	// ComponentNameFunc is used to get the component name if ComponentName
	// is empty.
	//
	// Default: "net/http"
	ComponentNameFunc func(*http.Request) string

	// URLTagFunc is used to get the value of the tag "http.url".
	//
	// Default: url.String()
	URLTagFunc func(*url.URL) string

	// SpanFilter is used to filter the span if returning true.
	//
	// Default: return false
	SpanFilter func(*http.Request) bool

	// OperationNameFunc is used to the operation name.
	//
	// Default: fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Path)
	OperationNameFunc func(*http.Request) string

	// SpanObserver is used to do extra things of the span for the request.
	//
	// For example,
	//    OpenTracingOption {
	//        SpanObserver: func(*http.Request, opentracing.Span) {
	//            ext.PeerHostname.Set(span, req.Host)
	//        },
	//    }
	//
	// Default: Do nothing.
	SpanObserver func(*http.Request, opentracing.Span)
}

// Init initializes the OpenTracingOption.
func (o *Option) Init() {
	if o.ComponentNameFunc == nil {
		o.ComponentNameFunc = func(*http.Request) string { return "net/http" }
	}
	if o.URLTagFunc == nil {
		o.URLTagFunc = func(u *url.URL) string { return u.String() }
	}
	if o.SpanFilter == nil {
		o.SpanFilter = func(r *http.Request) bool { return false }
	}
	if o.SpanObserver == nil {
		o.SpanObserver = func(*http.Request, opentracing.Span) {}
	}
	if o.OperationNameFunc == nil {
		o.OperationNameFunc = func(r *http.Request) string {
			return fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Path)
		}
	}
}

// GetComponentName returns ComponentName if it is not empty.
// Or ComponentNameFunc(req) instead.
func (o *Option) GetComponentName(req *http.Request) string {
	if o.ComponentName == "" {
		return o.ComponentNameFunc(req)
	}
	return o.ComponentName
}

// GetTracer returns the OpenTracing tracker.
func (o *Option) GetTracer() opentracing.Tracer {
	if o.Tracer != nil {
		return o.Tracer
	}
	return opentracing.GlobalTracer()
}
