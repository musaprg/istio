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
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"istio.io/istio/pkg/log"
)

// createOrUpdatePolicyUnstructured creates or updates a MutatingAdmissionPolicy using unstructured API
func (c *Controller) createOrUpdatePolicyUnstructured(ctx context.Context, policy *unstructured.Unstructured) error {
	log.Infof("DEBUG: createOrUpdatePolicyUnstructured called for policy: %s", policy.GetName())

	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(policy.GroupVersionKind())

	log.Info("DEBUG: About to call c.Get() for existing policy")
	err := c.Get(ctx, types.NamespacedName{Name: policy.GetName()}, existing)
	log.Infof("DEBUG: c.Get() completed with error: %v", err)

	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to get existing policy: %w", err)
		}
		// Policy doesn't exist, create it
		log.Infof("Creating MutatingAdmissionPolicy %s", policy.GetName())
		log.Info("DEBUG: About to call c.Create()")
		err := c.Create(ctx, policy)
		log.Infof("DEBUG: c.Create() completed with error: %v", err)
		return err
	}

	// Policy exists, update it
	log.Infof("Updating MutatingAdmissionPolicy %s", policy.GetName())
	log.Info("DEBUG: About to call c.Update()")

	// Update the spec and labels
	if spec, found, err := unstructured.NestedMap(policy.Object, "spec"); err == nil && found {
		if err := unstructured.SetNestedMap(existing.Object, spec, "spec"); err != nil {
			return fmt.Errorf("failed to set spec: %w", err)
		}
	}

	existing.SetLabels(policy.GetLabels())

	err = c.Update(ctx, existing)
	log.Infof("DEBUG: c.Update() completed with error: %v", err)
	return err
}

// createOrUpdatePolicyBindingUnstructured creates or updates a MutatingAdmissionPolicyBinding using unstructured API
func (c *Controller) createOrUpdatePolicyBindingUnstructured(ctx context.Context, binding *unstructured.Unstructured) error {
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(binding.GroupVersionKind())

	err := c.Get(ctx, types.NamespacedName{Name: binding.GetName()}, existing)

	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to get existing policy binding: %w", err)
		}
		// Binding doesn't exist, create it
		log.Infof("Creating MutatingAdmissionPolicyBinding %s", binding.GetName())
		return c.Create(ctx, binding)
	}

	// Binding exists, update it
	log.Infof("Updating MutatingAdmissionPolicyBinding %s", binding.GetName())

	// Update the spec and labels
	if spec, found, err := unstructured.NestedMap(binding.Object, "spec"); err == nil && found {
		if err := unstructured.SetNestedMap(existing.Object, spec, "spec"); err != nil {
			return fmt.Errorf("failed to set spec: %w", err)
		}
	}

	existing.SetLabels(binding.GetLabels())

	return c.Update(ctx, existing)
}

// ReconcileUnstructured handles reconciliation of MutatingAdmissionPolicy resources using unstructured API
func (c *Controller) ReconcileUnstructured(ctx context.Context) error {
	log.Info("Starting reconciliation using unstructured API")
	log.Info("DEBUG: Starting reconciliation - generating revision")

	// Generate new revision number
	newRevision := c.currentRevision + 1
	c.currentRevision = newRevision
	c.configMapName = fmt.Sprintf("%s-%d", ConfigMapNamePrefix, c.currentRevision)

	// Step 1: Generate injection configuration ConfigMap
	log.Info("DEBUG: Step 1 - Generating injection ConfigMap")
	configMap, err := c.generateInjectionConfigMap()
	if err != nil {
		log.Errorf("Failed to generate injection ConfigMap: %v", err)
		return err
	}
	log.Info("DEBUG: Step 1 completed successfully")

	// Step 2: Create or update ConfigMap
	log.Info("DEBUG: Step 2 - Creating/updating ConfigMap")
	err = c.createOrUpdateConfigMap(ctx, configMap)
	if err != nil {
		log.Errorf("Failed to create/update ConfigMap: %v", err)
		return err
	}
	log.Info("DEBUG: Step 2 completed successfully")

	// Step 3: Generate MutatingAdmissionPolicy resources using unstructured API
	log.Info("DEBUG: Step 3 - Generating MutatingAdmissionPolicy resources")
	policies, err := c.generatePoliciesUnstructured()
	if err != nil {
		log.Errorf("Failed to generate policies: %v", err)
		return err
	}
	log.Infof("DEBUG: Step 3 completed successfully, generated %d policies", len(policies))

	// Step 4: Create or update policies
	log.Info("DEBUG: Step 4 - Creating/updating policies")
	for i, policy := range policies {
		log.Infof("DEBUG: Creating/updating policy %d/%d: %s", i+1, len(policies), policy.GetName())
		err = c.createOrUpdatePolicyUnstructured(ctx, policy)
		if err != nil {
			log.Errorf("Failed to create/update policy %s: %v", policy.GetName(), err)
			return err
		}
		log.Infof("DEBUG: Successfully created/updated policy: %s", policy.GetName())
	}
	log.Info("DEBUG: Step 4 completed successfully")

	// Step 5: Create or update policy bindings
	log.Info("DEBUG: Step 5 - Generating policy bindings")
	bindings, err := c.generatePolicyBindingsUnstructured()
	if err != nil {
		log.Errorf("Failed to generate policy bindings: %v", err)
		return err
	}

	for _, binding := range bindings {
		err = c.createOrUpdatePolicyBindingUnstructured(ctx, binding)
		if err != nil {
			log.Errorf("Failed to create/update policy binding %s: %v", binding.GetName(), err)
			return err
		}
	}
	log.Info("DEBUG: Step 5 completed successfully")

	log.Infof("Successfully reconciled MutatingAdmissionPolicy resources with revision %d", c.currentRevision)
	return nil
}