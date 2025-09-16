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
	log.Info("DEBUG: generateBaseCELExpression called - implementing complete JSONPatch sidecar injection")
	// Complete JSONPatch approach with step-by-step container building
	// Each patch operation must be atomic and type-safe
	return `[
		JSONPatch{op: "add", path: "/metadata/labels/sidecar.istio.io~1inject", value: "true"},
		JSONPatch{op: "add", path: "/metadata/labels/istio.io~1rev", value: "default"},
		JSONPatch{op: "add", path: "/metadata/annotations/sidecar.istio.io~1interceptionMode", value: "REDIRECT"},
		JSONPatch{op: "add", path: "/metadata/annotations/traffic.sidecar.istio.io~1includeInboundPorts", value: "*"},
		JSONPatch{op: "add", path: "/metadata/annotations/traffic.sidecar.istio.io~1excludeInboundPorts", value: "15090,15021,15020"},
		JSONPatch{op: "add", path: "/spec/containers/-", value: {
			"name": "istio-proxy",
			"image": "gcr.io/istio-testing/proxyv2:latest",
			"args": ["proxy", "sidecar", "--domain", "$(POD_NAMESPACE).svc.cluster.local", "--proxyLogLevel=warning", "--proxyComponentLogLevel=misc:error", "--log_output_level=default:info"],
			"env": [
				{"name": "POD_NAME", "valueFrom": {"fieldRef": {"fieldPath": "metadata.name"}}},
				{"name": "POD_NAMESPACE", "valueFrom": {"fieldRef": {"fieldPath": "metadata.namespace"}}},
				{"name": "PILOT_CERT_PROVIDER", "value": "istiod"},
				{"name": "CA_ADDR", "value": "istiod.istio-system.svc:15012"},
				{"name": "ISTIO_META_CLUSTER_ID", "value": "Kubernetes"},
				{"name": "ISTIO_META_INTERCEPTION_MODE", "value": "REDIRECT"},
				{"name": "ISTIO_META_MESH_ID", "value": "cluster.local"},
				{"name": "TRUST_DOMAIN", "value": "cluster.local"}
			],
			"ports": [{"name": "http-envoy-prom", "containerPort": 15090, "protocol": "TCP"}],
			"securityContext": {
				"runAsUser": 1337,
				"runAsGroup": 1337,
				"runAsNonRoot": true,
				"readOnlyRootFilesystem": true,
				"allowPrivilegeEscalation": false,
				"capabilities": {"drop": ["ALL"]}
			},
			"resources": {
				"requests": {"cpu": "100m", "memory": "128Mi"},
				"limits": {"cpu": "2000m", "memory": "1Gi"}
			}
		}},
		JSONPatch{op: "add", path: "/spec/initContainers", value: [{
			"name": "istio-init",
			"image": "gcr.io/istio-testing/proxyv2:latest",
			"args": ["istio-iptables", "-p", "15001", "-z", "15006", "-u", "1337", "-m", "REDIRECT", "-i", "*", "-x", "", "-b", "*", "-d", "15090,15021,15020", "--log_output_level=default:info"],
			"securityContext": {
				"runAsUser": 0,
				"runAsGroup": 0,
				"runAsNonRoot": false,
				"readOnlyRootFilesystem": false,
				"allowPrivilegeEscalation": false,
				"capabilities": {"add": ["NET_ADMIN", "NET_RAW"], "drop": ["ALL"]}
			},
			"resources": {
				"requests": {"cpu": "100m", "memory": "128Mi"},
				"limits": {"cpu": "2000m", "memory": "1Gi"}
			}
		}]}
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