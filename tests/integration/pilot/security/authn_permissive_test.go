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

// Package basic contains an example test suite for showcase purposes.
package security

import (
	"reflect"
	"testing"
	"time"

	xdsapi "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	lis "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	proto "github.com/gogo/protobuf/types"

	authnv1alpha "istio.io/api/authentication/v1alpha1"
	"istio.io/istio/pilot/pkg/model"
	authnplugin "istio.io/istio/pilot/pkg/networking/plugin/authn"
	"istio.io/istio/pkg/test/framework"
	"istio.io/istio/pkg/test/framework/components/apps"
	"istio.io/istio/pkg/test/framework/components/environment"
	"istio.io/istio/pkg/test/framework/components/environment/native"
	pilot2 "istio.io/istio/pkg/test/framework/components/pilot"
)

// To opt-in to the test framework, implement a TestMain, and call test.Run.

func TestMain(m *testing.M) {
	framework.Main("authn_permissive_test", m)
}

func verifyListener(listener *xdsapi.Listener, t *testing.T) bool {
	t.Helper()
	if listener == nil {
		return false
	}
	if len(listener.ListenerFilters) == 0 {
		return false
	}
	// We expect tls_inspector filter exist.
	inspector := false
	for _, lf := range listener.ListenerFilters {
		if lf.Name == authnplugin.EnvoyTLSInspectorFilterName {
			inspector = true
			break
		}
	}
	if !inspector {
		return false
	}
	// Check filter chain match.
	if len(listener.FilterChains) != 2 {
		return false
	}
	mtlsChain := listener.FilterChains[0]
	if !reflect.DeepEqual(mtlsChain.FilterChainMatch.ApplicationProtocols, []string{"istio"}) {
		return false
	}
	if mtlsChain.TlsContext == nil {
		return false
	}
	// Second default filter chain should have empty filter chain match and no tls context.
	defaultChain := listener.FilterChains[1]
	if !reflect.DeepEqual(defaultChain.FilterChainMatch, &lis.FilterChainMatch{}) {
		return false
	}
	if defaultChain.TlsContext != nil {
		return false
	}
	return true
}

// TestAuthnPermissive checks when authentication policy is permissive, Pilot generates expected
// listener configuration.
func TestAuthnPermissive(t *testing.T) {
	ctx := framework.NewContext(t)
	defer ctx.Done(t)

	// TODO(incfly): make test able to run both on k8s and native when galley is ready.
	ctx.RequireOrSkip(t, environment.Native)

	env := ctx.Environment().(*native.Environment)
	_, err := env.ServiceManager.ConfigStore.Create(
		model.Config{
			ConfigMeta: model.ConfigMeta{
				Type:      model.AuthenticationPolicy.Type,
				Name:      "default",
				Namespace: "istio-system",
			},
			Spec: &authnv1alpha.Policy{
				// TODO: make policy work just applied to service a.
				// Targets: []*authn.TargetSelector{
				// 	{
				// 		Name: "a.istio-system.svc.local",
				// 	},
				// },
				Peers: []*authnv1alpha.PeerAuthenticationMethod{{
					Params: &authnv1alpha.PeerAuthenticationMethod_Mtls{
						Mtls: &authnv1alpha.MutualTls{
							Mode: authnv1alpha.MutualTls_PERMISSIVE,
						},
					},
				}},
			},
		},
	)
	if err != nil {
		t.Error(err)
	}
	pilot := pilot2.NewOrFail(t, ctx, pilot2.Config{})
	aps := apps.NewOrFail(ctx, t, apps.Config{Pilot: pilot})
	a := aps.GetAppOrFail("a", t)
	req := apps.ConstructDiscoveryRequest(a, "type.googleapis.com/envoy.api.v2.Listener")
	err = pilot.StartDiscovery(req)
	if err != nil {
		t.Fatalf("failed to call discovery %v", err)
	}
	err = pilot.WatchDiscovery(time.Second*10,
		func(resp *xdsapi.DiscoveryResponse) (b bool, e error) {
			for _, r := range resp.Resources {
				foo := &xdsapi.Listener{}
				err := proto.UnmarshalAny(&r, foo)
				result := verifyListener(foo, t)
				if err == nil && result {
					return true, nil
				}
			}
			return false, nil
		})
	if err != nil {
		t.Fatalf("failed to find any listeners having multiplexing filter chain : %v", err)
	}
}

// TestAuthentictionPermissiveE2E these cases are covered end to end
// app A to app B using plaintext (mTLS),
// app A to app B using HTTPS (mTLS),
// app A to app B using plaintext (legacy),
// app A to app B using HTTPS (legacy).
// explained: app-to-app-protocol(sidecar-to-sidecar-protocol). "legacy" means
// no client sidecar, unable to send "istio" alpn indicator.
// TODO(incfly): implement this
// func TestAuthentictionPermissiveE2E(t *testing.T) {
// Steps:
// Configure authn policy.
// Wait for config propagation.
// Send HTTP requests between apps.
// }
