// Copyright Istio Authors
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

package policy

import (
	"istio.io/istio/pkg/log"
)

// generatePolicies creates the MutatingAdmissionPolicy resources (DEPRECATED - use generatePoliciesUnstructured)
func (c *Controller) generatePolicies() (interface{}, error) {
	log.Info("DEPRECATED: generatePolicies called - use generatePoliciesUnstructured instead")
	return nil, nil
}

// generateBaseSidecarPolicy creates the base sidecar injection policy (DEPRECATED - use generateBaseSidecarPolicyUnstructured)
func (c *Controller) generateBaseSidecarPolicy() interface{} {
	log.Info("DEPRECATED: generateBaseSidecarPolicy called - use generateBaseSidecarPolicyUnstructured instead")
	return nil
}

// generateBaseCELExpression creates the CEL expression for base sidecar injection using JSONPatch
func (c *Controller) generateBaseCELExpression() string {
	log.Info("DEBUG: generateBaseCELExpression called - implementing incremental JSONPatch sidecar injection")
	// BREAKTHROUGH SOLUTION: Incremental JSONPatch approach overcomes CEL array limitations
	// Build complex containers step-by-step using separate operations for each array/object field
	return `[
		JSONPatch{op: "add", path: "/metadata/labels/sidecar.istio.io~1inject", value: "true"},
		JSONPatch{op: "add", path: "/metadata/labels/istio.io~1rev", value: "default"},
		JSONPatch{op: "add", path: "/metadata/annotations/sidecar.istio.io~1interceptionMode", value: "REDIRECT"},
		JSONPatch{op: "add", path: "/metadata/annotations/traffic.sidecar.istio.io~1includeInboundPorts", value: "*"},
		JSONPatch{op: "add", path: "/metadata/annotations/traffic.sidecar.istio.io~1excludeInboundPorts", value: "15090,15021,15020"},

		JSONPatch{op: "add", path: "/spec/containers/-", value: {"name": "istio-proxy", "image": "gcr.io/istio-testing/proxyv2:latest"}},
		JSONPatch{op: "add", path: "/spec/containers/1/command", value: ["/usr/local/bin/pilot-agent"]},
		JSONPatch{op: "add", path: "/spec/containers/1/args", value: ["proxy", "sidecar", "--domain", "$(POD_NAMESPACE).svc.cluster.local", "--proxyLogLevel=warning", "--proxyComponentLogLevel=misc:error", "--log_output_level=default:info"]},

		JSONPatch{op: "add", path: "/spec/containers/1/env", value: []},
		JSONPatch{op: "add", path: "/spec/containers/1/env/-", value: {"name": "POD_NAME"}},
		JSONPatch{op: "add", path: "/spec/containers/1/env/0/valueFrom", value: {"fieldRef": {"fieldPath": "metadata.name"}}},
		JSONPatch{op: "add", path: "/spec/containers/1/env/-", value: {"name": "POD_NAMESPACE"}},
		JSONPatch{op: "add", path: "/spec/containers/1/env/1/valueFrom", value: {"fieldRef": {"fieldPath": "metadata.namespace"}}},
		JSONPatch{op: "add", path: "/spec/containers/1/env/-", value: {"name": "PILOT_CERT_PROVIDER", "value": "istiod"}},
		JSONPatch{op: "add", path: "/spec/containers/1/env/-", value: {"name": "CA_ADDR", "value": "istiod.istio-system.svc:15012"}},
		JSONPatch{op: "add", path: "/spec/containers/1/env/-", value: {"name": "ISTIO_META_CLUSTER_ID", "value": "Kubernetes"}},
		JSONPatch{op: "add", path: "/spec/containers/1/env/-", value: {"name": "TRUST_DOMAIN", "value": "cluster.local"}},

		JSONPatch{op: "add", path: "/spec/containers/1/ports", value: []},
		JSONPatch{op: "add", path: "/spec/containers/1/ports/-", value: {"name": "http-envoy-prom", "containerPort": 15090, "protocol": "TCP"}},

		JSONPatch{op: "add", path: "/spec/containers/1/securityContext", value: {"runAsUser": 1337, "runAsGroup": 1337, "runAsNonRoot": true, "readOnlyRootFilesystem": true, "allowPrivilegeEscalation": false}},

		JSONPatch{op: "add", path: "/spec/initContainers", value: [{"name": "istio-init", "image": "gcr.io/istio-testing/proxyv2:latest"}]},
		JSONPatch{op: "add", path: "/spec/initContainers/0/command", value: ["/usr/local/bin/pilot-agent"]},
		JSONPatch{op: "add", path: "/spec/initContainers/0/args", value: ["istio-iptables", "-p", "15001", "-z", "15006", "-u", "1337", "-m", "REDIRECT", "-i", "*", "-x", "", "-b", "*", "-d", "15090,15021,15020"]},
		JSONPatch{op: "add", path: "/spec/initContainers/0/securityContext", value: {"runAsUser": 0, "runAsGroup": 0, "runAsNonRoot": false, "allowPrivilegeEscalation": false}}
	]`
}

// generatePolicyBindings creates the MutatingAdmissionPolicyBinding resources (DEPRECATED - use generatePolicyBindingsUnstructured)
func (c *Controller) generatePolicyBindings() (interface{}, error) {
	log.Info("DEPRECATED: generatePolicyBindings called - use generatePolicyBindingsUnstructured instead")
	return nil, nil
}

// generateBaseSidecarPolicyBinding creates the policy binding for base sidecar injection (DEPRECATED - use generateBaseSidecarPolicyBindingUnstructured)
func (c *Controller) generateBaseSidecarPolicyBinding() interface{} {
	log.Info("DEPRECATED: generateBaseSidecarPolicyBinding called - use generateBaseSidecarPolicyBindingUnstructured instead")
	return nil
}