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

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"istio.io/istio/pkg/log"
)

// generatePoliciesUnstructured creates the MutatingAdmissionPolicy resources using unstructured API
func (c *Controller) generatePoliciesUnstructured() ([]*unstructured.Unstructured, error) {
	log.Info("Generating MutatingAdmissionPolicy resources using unstructured API")

	policies := []*unstructured.Unstructured{
		c.generateBaseSidecarPolicyUnstructured(),
	}

	return policies, nil
}

// generateBaseSidecarPolicyUnstructured creates the base sidecar injection policy
func (c *Controller) generateBaseSidecarPolicyUnstructured() *unstructured.Unstructured {
	policyName := fmt.Sprintf("%s-base", PolicyNamePrefix)

	celExpression := c.generateBaseCELExpression()

	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "admissionregistration.k8s.io/v1beta1",
			"kind":       "MutatingAdmissionPolicy",
			"metadata": map[string]interface{}{
				"name": policyName,
				"labels": map[string]interface{}{
					"istio.io/policy-group":       "sidecar-injection",
					"istio.io/policy-type":        "base",
					"istio.io/revision":           c.revision,
					"app.kubernetes.io/managed-by": PolicyControllerName,
				},
			},
			"spec": map[string]interface{}{
				"paramKind": map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
				},
				"matchConstraints": map[string]interface{}{
					"resourceRules": []interface{}{
						map[string]interface{}{
							"operations": []interface{}{"CREATE"},
							"apiGroups":  []interface{}{""},
							"apiVersions": []interface{}{"v1"},
							"resources":   []interface{}{"pods"},
						},
					},
					"objectSelector": map[string]interface{}{
						"matchExpressions": []interface{}{
							map[string]interface{}{
								"key":      "sidecar.istio.io/inject",
								"operator": "NotIn",
								"values":   []interface{}{"false"},
							},
						},
					},
					"namespaceSelector": map[string]interface{}{
						"matchLabels": map[string]interface{}{
							"injection-method": "policy",
							"istio-injection":  "enabled",
						},
					},
				},
				"reinvocationPolicy": "Never",
				"mutations": []interface{}{
					map[string]interface{}{
						"patchType": "JSONPatch",
						"jsonPatch": map[string]interface{}{
							"expression": celExpression,
						},
					},
				},
			},
		},
	}

	return policy
}

// generatePolicyBindingsUnstructured creates the MutatingAdmissionPolicyBinding resources using unstructured API
func (c *Controller) generatePolicyBindingsUnstructured() ([]*unstructured.Unstructured, error) {
	log.Info("Generating MutatingAdmissionPolicyBinding resources using unstructured API")

	bindings := []*unstructured.Unstructured{
		c.generateBaseSidecarPolicyBindingUnstructured(),
	}

	return bindings, nil
}

// generateBaseSidecarPolicyBindingUnstructured creates the policy binding for base sidecar injection
func (c *Controller) generateBaseSidecarPolicyBindingUnstructured() *unstructured.Unstructured {
	policyName := fmt.Sprintf("%s-base", PolicyNamePrefix)
	bindingName := fmt.Sprintf("%s-base-binding", PolicyNamePrefix)

	binding := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "admissionregistration.k8s.io/v1beta1",
			"kind":       "MutatingAdmissionPolicyBinding",
			"metadata": map[string]interface{}{
				"name": bindingName,
				"labels": map[string]interface{}{
					"istio.io/policy-group":       "sidecar-injection",
					"istio.io/policy-type":        "base",
					"istio.io/revision":           c.revision,
					"app.kubernetes.io/managed-by": PolicyControllerName,
				},
			},
			"spec": map[string]interface{}{
				"policyName": policyName,
				"paramRef": map[string]interface{}{
					"name":      c.configMapName,
					"namespace": c.namespace,
					"parameterNotFoundAction": "Allow",
				},
				"matchResources": map[string]interface{}{
					"resourceRules": []interface{}{
						map[string]interface{}{
							"operations": []interface{}{"CREATE"},
							"apiGroups":  []interface{}{""},
							"apiVersions": []interface{}{"v1"},
							"resources":   []interface{}{"pods"},
						},
					},
					"objectSelector": map[string]interface{}{
						"matchExpressions": []interface{}{
							map[string]interface{}{
								"key":      "sidecar.istio.io/inject",
								"operator": "NotIn",
								"values":   []interface{}{"false"},
							},
						},
					},
					"namespaceSelector": map[string]interface{}{
						"matchLabels": map[string]interface{}{
							"istio-injection":  "enabled",
							"injection-method": "policy",
						},
					},
				},
			},
		},
	}

	return binding
}