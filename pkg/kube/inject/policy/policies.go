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

	admissionregistrationv1alpha1 "k8s.io/api/admissionregistration/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"istio.io/istio/pkg/log"
)

// generatePolicies creates the MutatingAdmissionPolicy resources
func (c *Controller) generatePolicies() ([]*admissionregistrationv1alpha1.MutatingAdmissionPolicy, error) {
	log.Info("Generating MutatingAdmissionPolicy resources")

	policies := []*admissionregistrationv1alpha1.MutatingAdmissionPolicy{
		c.generateBaseSidecarPolicy(),
	}

	return policies, nil
}

// generateBaseSidecarPolicy creates the base sidecar injection policy
func (c *Controller) generateBaseSidecarPolicy() *admissionregistrationv1alpha1.MutatingAdmissionPolicy {
	policyName := fmt.Sprintf("%s-base", PolicyNamePrefix)
	
	celExpression := c.generateBaseCELExpression()

	return &admissionregistrationv1alpha1.MutatingAdmissionPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: policyName,
			Labels: map[string]string{
				"istio.io/policy-group":       "sidecar-injection",
				"istio.io/policy-type":        "base",
				"istio.io/revision":           c.revision,
				"app.kubernetes.io/managed-by": PolicyControllerName,
			},
		},
		Spec: admissionregistrationv1alpha1.MutatingAdmissionPolicySpec{
			ParamKind: &admissionregistrationv1alpha1.ParamKind{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			MatchConstraints: &admissionregistrationv1alpha1.MatchResources{
				ResourceRules: []admissionregistrationv1alpha1.NamedRuleWithOperations{
					{
						RuleWithOperations: admissionregistrationv1alpha1.RuleWithOperations{
							Operations: []admissionregistrationv1alpha1.OperationType{
								admissionregistrationv1alpha1.Create,
							},
							Rule: admissionregistrationv1alpha1.Rule{
								APIGroups:   []string{""},
								APIVersions: []string{"v1"},
								Resources:   []string{"pods"},
							},
						},
					},
				},
				ObjectSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "sidecar.istio.io/inject",
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{"false"},
						},
					},
				},
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"injection-method": "policy",
					},
				},
			},
			Mutations: []admissionregistrationv1alpha1.Mutation{
				{
					PatchType: admissionregistrationv1alpha1.PatchTypeApplyConfiguration,
					ApplyConfiguration: &admissionregistrationv1alpha1.ApplyConfiguration{
						Expression: celExpression,
					},
				},
			},
		},
	}
}

// generateBaseCELExpression creates the CEL expression for base sidecar injection
func (c *Controller) generateBaseCELExpression() string {
	// This is the core CEL expression that applies pre-calculated values
	// and handles dynamic Pod-specific data
	return fmt.Sprintf(`
Object{
  metadata: Object.metadata{
    labels: object.metadata.labels + {
      "security.istio.io/tlsMode": "istio",
      "service.istio.io/canonical-name": has(object.metadata.labels.app) ? 
        object.metadata.labels.app : 
        (has(object.metadata.labels["app.kubernetes.io/name"]) ? 
         object.metadata.labels["app.kubernetes.io/name"] : "unknown"),
      "service.istio.io/canonical-revision": has(object.metadata.labels.version) ?
        object.metadata.labels.version : "latest"
    },
    annotations: object.metadata.annotations + 
      (has(params) && has(params.data) && has(params.data["calculated-annotations.json"]) ?
        params.data["calculated-annotations.json"] : {}) + {
      "kubectl.kubernetes.io/default-container": size(object.spec.containers) > 0 ? 
        object.spec.containers[0].name : "",
      "kubectl.kubernetes.io/default-logs-container": size(object.spec.containers) > 0 ? 
        object.spec.containers[0].name : "",
      "sidecar.istio.io/status": '{"initContainers":["istio-init","istio-proxy"],"containers":null,"volumes":["workload-socket","credential-socket","workload-certs","istio-envoy","istio-data","istio-podinfo","istio-token","istiod-ca-cert","istio-ca-crl"],"imagePullSecrets":null,"revision":"%s"}'
    }
  },
  spec: Object.spec{
    initContainers: object.spec.initContainers + [
      (has(params) && has(params.data) && has(params.data["init-container.json"]) ?
        params.data["init-container.json"] : {
          "name": "istio-init",
          "image": "gcr.io/istio-testing/proxyv2:latest"
        })
    ],
    containers: object.spec.containers + [
      (has(params) && has(params.data) && has(params.data["sidecar-container.json"]) ?
        params.data["sidecar-container.json"] + {
          env: (has(params.data["sidecar-container.json"].env) ? 
            params.data["sidecar-container.json"].env : []) + 
            (has(params) && has(params.data) && has(params.data["static-env-vars.json"]) ?
              params.data["static-env-vars.json"] : []) + [
            {name: "POD_NAME", valueFrom: {fieldRef: {fieldPath: "metadata.name"}}},
            {name: "POD_NAMESPACE", valueFrom: {fieldRef: {fieldPath: "metadata.namespace"}}},
            {name: "INSTANCE_IP", valueFrom: {fieldRef: {fieldPath: "status.podIP"}}},
            {name: "SERVICE_ACCOUNT", valueFrom: {fieldRef: {fieldPath: "spec.serviceAccountName"}}},
            {name: "HOST_IP", valueFrom: {fieldRef: {fieldPath: "status.hostIP"}}},
            {name: "ISTIO_CPU_LIMIT", valueFrom: {resourceFieldRef: {resource: "limits.cpu"}}},
            {name: "GOMEMLIMIT", valueFrom: {resourceFieldRef: {resource: "limits.memory"}}},
            {name: "GOMAXPROCS", valueFrom: {resourceFieldRef: {resource: "limits.cpu"}}},
            {name: "ISTIO_META_POD_PORTS", value: 
              "[" + object.spec.containers.map(c, 
                c.ports.map(p, '{"name":"' + (has(p.name) ? p.name : "") + '","containerPort":' + string(p.containerPort) + '}')
              ).flatten().join(",") + "]"},
            {name: "ISTIO_META_APP_CONTAINERS", value: 
              object.spec.containers.filter(c, c.name != "istio-proxy").map(c, c.name).join(",")},
            {name: "ISTIO_META_WORKLOAD_NAME", value: 
              has(object.metadata.labels.app) ? object.metadata.labels.app : object.metadata.name},
            {name: "ISTIO_META_OWNER", value: 
              "kubernetes://apis/apps/v1/namespaces/" + object.metadata.namespace + "/deployments/" + object.metadata.name},
            {name: "ISTIO_META_NODE_NAME", valueFrom: {fieldRef: {fieldPath: "spec.nodeName"}}}
          ]
        } : {
          "name": "istio-proxy",
          "image": "gcr.io/istio-testing/proxyv2:latest"
        })
    ],
    volumes: object.spec.volumes + 
      (has(params) && has(params.data) && has(params.data["base-volumes.json"]) ?
        params.data["base-volumes.json"] : [])
  }
}`, c.revision)
}

// generatePolicyBindings creates the MutatingAdmissionPolicyBinding resources
func (c *Controller) generatePolicyBindings() ([]*admissionregistrationv1alpha1.MutatingAdmissionPolicyBinding, error) {
	log.Info("Generating MutatingAdmissionPolicyBinding resources")

	bindings := []*admissionregistrationv1alpha1.MutatingAdmissionPolicyBinding{
		c.generateBaseSidecarPolicyBinding(),
	}

	return bindings, nil
}

// generateBaseSidecarPolicyBinding creates the policy binding for base sidecar injection
func (c *Controller) generateBaseSidecarPolicyBinding() *admissionregistrationv1alpha1.MutatingAdmissionPolicyBinding {
	policyName := fmt.Sprintf("%s-base", PolicyNamePrefix)
	bindingName := fmt.Sprintf("%s-base-binding", PolicyNamePrefix)

	return &admissionregistrationv1alpha1.MutatingAdmissionPolicyBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: bindingName,
			Labels: map[string]string{
				"istio.io/policy-group":       "sidecar-injection",
				"istio.io/policy-type":        "base",
				"istio.io/revision":           c.revision,
				"app.kubernetes.io/managed-by": PolicyControllerName,
			},
		},
		Spec: admissionregistrationv1alpha1.MutatingAdmissionPolicyBindingSpec{
			PolicyName: policyName,
			ParamRef: &admissionregistrationv1alpha1.ParamRef{
				Name:      c.configMapName,
				Namespace: c.namespace,
			},
			// ValidationActions is not part of MutatingAdmissionPolicyBindingSpec
			// Enforcement is implicit for MutatingAdmissionPolicy
			MatchResources: &admissionregistrationv1alpha1.MatchResources{
				ResourceRules: []admissionregistrationv1alpha1.NamedRuleWithOperations{
					{
						RuleWithOperations: admissionregistrationv1alpha1.RuleWithOperations{
							Operations: []admissionregistrationv1alpha1.OperationType{
								admissionregistrationv1alpha1.Create,
							},
							Rule: admissionregistrationv1alpha1.Rule{
								APIGroups:   []string{""},
								APIVersions: []string{"v1"},
								Resources:   []string{"pods"},
							},
						},
					},
				},
				ObjectSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "sidecar.istio.io/inject",
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{"false"},
						},
					},
				},
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"istio-injection":  "enabled",
						"injection-method": "policy",
					},
				},
			},
		},
	}
}