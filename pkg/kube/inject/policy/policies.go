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
	"fmt"


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

// generateBaseCELExpression creates the CEL expression for base sidecar injection
func (c *Controller) generateBaseCELExpression() string {
	// This is a simplified CEL expression that adds basic sidecar injection
	// without complex parameter handling to avoid CEL syntax issues
	return fmt.Sprintf(`
Object{
  metadata: Object.metadata{
    labels: {
      "security.istio.io/tlsMode": "istio",
      "service.istio.io/canonical-name": has(object.metadata.labels) && "app" in object.metadata.labels ?
        object.metadata.labels["app"] : "unknown",
      "service.istio.io/canonical-revision": has(object.metadata.labels) && "version" in object.metadata.labels ?
        object.metadata.labels["version"] : "latest"
    },
    annotations: {
      "kubectl.kubernetes.io/default-container": size(object.spec.containers) > 0 ?
        object.spec.containers[0].name : "",
      "kubectl.kubernetes.io/default-logs-container": size(object.spec.containers) > 0 ?
        object.spec.containers[0].name : "",
      "sidecar.istio.io/status": '{"initContainers":["istio-init"],"containers":["istio-proxy"],"volumes":[],"imagePullSecrets":null,"revision":"%s"}'
    }
  },
  spec: Object.spec{
    initContainers: [{
      "name": "istio-init",
      "image": "localhost:5001/proxyv2:latest",
      "args": ["istio-iptables", "-p", "15001", "-z", "15006", "-u", "1337", "-m", "REDIRECT", "-i", "*", "-x", "", "-b", "*", "-d", "15090,15021,15020"],
      "securityContext": {
        "allowPrivilegeEscalation": false,
        "capabilities": {"add": ["NET_ADMIN", "NET_RAW"], "drop": ["ALL"]},
        "privileged": false,
        "readOnlyRootFilesystem": false,
        "runAsGroup": "0",
        "runAsNonRoot": false,
        "runAsUser": "0"
      }
    }],
    containers: object.spec.containers + [{
      "name": "istio-proxy",
      "image": "localhost:5001/proxyv2:latest",
      "args": ["proxy", "sidecar", "--domain", "$(POD_NAMESPACE).svc.cluster.local", "--proxyLogLevel=warning", "--proxyComponentLogLevel=misc:error"],
      "ports": [{"containerPort": 15090, "protocol": "TCP", "name": "http-envoy-prom"}],
      "env": [
        {"name": "POD_NAME", "valueFrom": {"fieldRef": {"fieldPath": "metadata.name"}}},
        {"name": "POD_NAMESPACE", "valueFrom": {"fieldRef": {"fieldPath": "metadata.namespace"}}},
        {"name": "INSTANCE_IP", "valueFrom": {"fieldRef": {"fieldPath": "status.podIP"}}},
        {"name": "SERVICE_ACCOUNT", "valueFrom": {"fieldRef": {"fieldPath": "spec.serviceAccountName"}}},
        {"name": "HOST_IP", "valueFrom": {"fieldRef": {"fieldPath": "status.hostIP"}}},
        {"name": "ISTIO_META_WORKLOAD_NAME", "value": has(object.metadata.labels) && "app" in object.metadata.labels ? object.metadata.labels["app"] : object.metadata.name},
        {"name": "ISTIO_META_OWNER", "value": "kubernetes://apis/apps/v1/namespaces/" + object.metadata.namespace + "/deployments/" + object.metadata.name}
      ],
      "securityContext": {
        "allowPrivilegeEscalation": false,
        "capabilities": {"drop": ["ALL"]},
        "privileged": false,
        "readOnlyRootFilesystem": true,
        "runAsUser": "1337",
        "runAsGroup": "1337",
        "runAsNonRoot": true
      }
    }]
  }
}`, c.revision)
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