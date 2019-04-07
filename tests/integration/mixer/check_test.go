//  Copyright 2018 Istio Authors
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package mixer

import (
	"fmt"
	"testing"
	"time"

	"istio.io/istio/pkg/test/framework"
	"istio.io/istio/pkg/test/framework/components/environment"
	"istio.io/istio/pkg/test/framework/components/galley"
	"istio.io/istio/pkg/test/framework/components/mixer"
	"istio.io/istio/pkg/test/framework/components/namespace"
	"istio.io/istio/pkg/test/framework/components/policybackend"
	"istio.io/istio/pkg/test/util/retry"
)

func TestCheck_Allow(t *testing.T) {
	framework.Run(t, func(ctx framework.TestContext) {
		// TODO(https://github.com/istio/istio/issues/12750): Disabling K8s mode for now, as Mixer is not listening for
		// legacy CRDs anymore, when using the standard Helm deployment.
		ctx.RequireOrSkip(t, environment.Native)

		gal := galley.NewOrFail(t, ctx, galley.Config{})
		mxr := mixer.NewOrFail(t, ctx, mixer.Config{
			Galley: gal,
		})
		be := policybackend.NewOrFail(t, ctx)

		ns := namespace.NewOrFail(t, ctx, "testcheck-allow", false)

		gal.ApplyConfigOrFail(
			t,
			ns,
			testCheckConfig,
			be.CreateConfigSnippet("handler1", ns.Name()))

		// Prime the policy backend'ctx behavior. It should deny all check requests.
		// This is not strictly necessary, but it is done so for posterity.
		be.DenyCheck(t, false)

		retry.UntilSuccessOrFail(t, func() error {
			result := mxr.Check(t, map[string]interface{}{
				"context.protocol":      "http",
				"destination.name":      "somesrvcname",
				"destination.namespace": ns.Name(),
				"response.time":         time.Now(),
				"request.time":          time.Now(),
				"destination.service":   `svc.` + ns.Name(),
				"origin.ip":             []byte{1, 2, 3, 4},
			})

			// TODO: ensure that the policy backend receives the request.
			if !result.Succeeded() {
				return fmt.Errorf("check failed: %v", result.Raw)
			}

			return nil
		}, retry.Timeout(time.Second*40))
	})
}

func TestCheck_Deny(t *testing.T) {
	framework.Run(t, func(ctx framework.TestContext) {
		// TODO(https://github.com/istio/istio/issues/12750): Disabling K8s mode for now, as Mixer is not listening for
		// legacy CRDs anymore, when using the standard Helm deployment.
		ctx.RequireOrSkip(t, environment.Native)

		gal := galley.NewOrFail(t, ctx, galley.Config{})
		mxr := mixer.NewOrFail(t, ctx, mixer.Config{
			Galley: gal,
		})
		be := policybackend.NewOrFail(t, ctx)

		ns := namespace.NewOrFail(t, ctx, "testcheck-deny", false)

		gal.ApplyConfigOrFail(
			t,
			ns,
			testCheckConfig,
			be.CreateConfigSnippet("handler1", ns.Name()))

		// Prime the policy backend'ctx behavior. It should deny all check requests.
		// This is not strictly necessary, but it is done so for posterity.
		be.DenyCheck(t, true)

		retry.UntilSuccessOrFail(t, func() error {
			result := mxr.Check(t, map[string]interface{}{
				"context.protocol":      "http",
				"destination.name":      "somesrvcname",
				"destination.namespace": ns.Name(),
				"response.time":         time.Now(),
				"request.time":          time.Now(),
				"destination.service":   `svc.` + ns.Name(),
				"origin.ip":             []byte{1, 2, 3, 4},
			})
			if result.Succeeded() {
				return fmt.Errorf("check failed: %v", result.Raw)
			}

			// TODO: ensure that the policy backend receives the request.

			return nil
		}, retry.Timeout(time.Second*40))
	})
}

var testCheckConfig = `
apiVersion: "config.istio.io/v1alpha2"
kind: checknothing
metadata:
  name: checknothing1
spec:
---
apiVersion: "config.istio.io/v1alpha2"
kind: rule
metadata:
  name: rule1
spec:
  actions:
  - handler: handler1.bypass
    instances:
    - checknothing1.checknothing
`
