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

package middlewares

import "net/http"

// Middleware is a http handler middleware.
type Middleware func(wrappedNext http.Handler) (new http.Handler)

// Handler is the extended http.Handler, which supports to return the error.
type Handler interface {
	HandleHTTP(w http.ResponseWriter, r *http.Request) error
	http.Handler
}
