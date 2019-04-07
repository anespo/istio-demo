// Copyright 2019 Istio Authors
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

package apps

import (
	"net/http"
	"testing"

	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/test/application/echo"
	"istio.io/istio/pkg/test/framework/components/environment"
	"istio.io/istio/pkg/test/framework/components/namespace"
	"istio.io/istio/pkg/test/framework/components/pilot"
	"istio.io/istio/pkg/test/framework/resource"
)

// AppProtocol enumerates the protocol options for calling an DeployedAppEndpoint endpoint.
type AppProtocol string

const (
	// AppProtocolHTTP calls the app with HTTP
	AppProtocolHTTP = "http"
	// AppProtocolGRPC calls the app with GRPC
	AppProtocolGRPC = "grpc"
	// AppProtocolWebSocket calls the app with WebSocket
	AppProtocolWebSocket = "ws"
)

// Instance is a component that provides access to all deployed test services.
type Instance interface {
	resource.Resource

	Namespace() namespace.Instance

	GetApp(name string) (App, error)
	GetAppOrFail(name string, t testing.TB) App
}

// App represents a deployed App within the mesh.
type App interface {
	Name() string
	Endpoints() []AppEndpoint
	EndpointsForProtocol(protocol model.Protocol) []AppEndpoint
	Call(e AppEndpoint, opts AppCallOptions) ([]*echo.ParsedResponse, error)
	CallOrFail(e AppEndpoint, opts AppCallOptions, t testing.TB) []*echo.ParsedResponse
}

// AppParam specifies the parameter for a single app.
type AppParam struct {
	Name string
}

// Config for Apps
type Config struct {
	Pilot pilot.Instance
	// AppParams specifies the apps needed for the test. If unspecified, a default suite of test apps
	// will be deployed.
	AppParams []AppParam
}

// AppCallOptions defines options for calling a DeployedAppEndpoint.
type AppCallOptions struct {
	// Protocol indicates the protocol to be used.
	Protocol AppProtocol

	// Count indicates the number of exchanges that should be made with the service endpoint. If not set (i.e. 0), defaults to 1.
	Count int

	// Headers indicates headers that should be sent in the request. Ingnored for WebSocket calls.
	Headers http.Header

	// Secure indicates whether a secure connection should be established to the endpoint.
	Secure bool

	// UseShortHostname indicates whether shortened hostnames should be used. This may be ignored by the environment.
	UseShortHostname bool
}

// AppEndpoint represents a single endpoint in a DeployedApp.
type AppEndpoint interface {
	Name() string
	Owner() App
	Protocol() model.Protocol
}

// New returns a new instance of Apps
func New(ctx resource.Context, cfg Config) (i Instance, err error) {
	err = resource.UnsupportedEnvironment(ctx.Environment())

	ctx.Environment().Case(environment.Native, func() {
		i, err = newNative(ctx, cfg)
	})
	ctx.Environment().Case(environment.Kube, func() {
		i, err = newKube(ctx, cfg)
	})

	return
}

// New returns a new instance of Apps or fails test.
func NewOrFail(ctx resource.Context, t *testing.T, cfg Config) Instance {
	t.Helper()

	i, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("apps.NewOrFail: %v", err)
	}

	return i
}
