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

// generateBaseCELExpression creates the CEL expression for base sidecar injection using ApplyConfiguration
func (c *Controller) generateBaseCELExpression() string {
	log.Info("DEBUG: generateBaseCELExpression called - implementing step-by-step sidecar injection")
	// ApplyConfiguration expects a CEL expression that returns an Object
	// Start with simpler injection to avoid CEL syntax complexity
	return `Object{
		metadata: Object.metadata{
			labels: {
				"sidecar.istio.io/inject": "true",
				"istio.io/rev": "default"
			},
			annotations: {
				"sidecar.istio.io/interceptionMode": "REDIRECT",
				"traffic.sidecar.istio.io/includeInboundPorts": "*",
				"traffic.sidecar.istio.io/excludeInboundPorts": "15090,15021,15020"
			}
		},
		spec: Object.spec{
			containers: [Object{
				name: "istio-proxy",
				image: "gcr.io/istio-testing/proxyv2:latest",
				args: [
					"proxy",
					"sidecar",
					"--domain",
					"$(POD_NAMESPACE).svc.cluster.local",
					"--proxyLogLevel=warning",
					"--proxyComponentLogLevel=misc:error",
					"--log_output_level=default:info"
				],
				ports: [Object{
					name: "http-envoy-prom",
					containerPort: 15090,
					protocol: "TCP"
				}],
				env: [
					Object{name: "POD_NAME", valueFrom: Object{fieldRef: Object{fieldPath: "metadata.name"}}},
					Object{name: "POD_NAMESPACE", valueFrom: Object{fieldRef: Object{fieldPath: "metadata.namespace"}}},
					Object{name: "PILOT_CERT_PROVIDER", value: "istiod"},
					Object{name: "CA_ADDR", value: "istiod.istio-system.svc:15012"}
				],
				resources: Object{
					requests: {
						"cpu": "100m",
						"memory": "128Mi"
					},
					limits: {
						"cpu": "2",
						"memory": "1Gi"
					}
				},
				securityContext: Object{
					runAsUser: 1337,
					runAsGroup: 1337,
					runAsNonRoot: true,
					readOnlyRootFilesystem: true,
					allowPrivilegeEscalation: false,
					capabilities: Object{
						drop: ["ALL"]
					}
				}
			}],
			initContainers: [Object{
				name: "istio-init",
				image: "gcr.io/istio-testing/proxyv2:latest",
				args: [
					"istio-iptables",
					"-p", "15001",
					"-z", "15006",
					"-u", "1337",
					"-m", "REDIRECT",
					"-i", "*",
					"-x", "",
					"-b", "*",
					"-d", "15090,15021,15020",
					"--log_output_level=default:info"
				],
				resources: Object{
					requests: {
						"cpu": "100m",
						"memory": "128Mi"
					},
					limits: {
						"cpu": "2",
						"memory": "1Gi"
					}
				},
				securityContext: Object{
					runAsUser: 0,
					runAsGroup: 0,
					runAsNonRoot: false,
					readOnlyRootFilesystem: false,
					allowPrivilegeEscalation: false,
					capabilities: Object{
						add: ["NET_ADMIN", "NET_RAW"],
						drop: ["ALL"]
					}
				}
			}]
		}
	}`
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