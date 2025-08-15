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

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"istio.io/istio/pkg/config/mesh"
	"istio.io/istio/pkg/kube/inject"
	"istio.io/istio/pkg/log"
)

// PolicyManager manages the MutatingAdmissionPolicy-based injection system
type PolicyManager struct {
	controller      *Controller
	meshWatcher     mesh.Watcher
	injectionConfig *inject.Config
	enabled         bool
}

// NewPolicyManager creates a new PolicyManager instance
func NewPolicyManager(mgr manager.Manager, meshWatcher mesh.Watcher, injectionConfig *inject.Config, namespace, revision string, values map[string]any) (*PolicyManager, error) {
	controller, err := NewController(mgr, meshWatcher, injectionConfig, namespace, revision, values)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy controller: %w", err)
	}

	return &PolicyManager{
		controller:      controller,
		meshWatcher:     meshWatcher,
		injectionConfig: injectionConfig,
		enabled:         false,
	}, nil
}

// Start initializes and starts the policy-based injection system
func (pm *PolicyManager) Start(ctx context.Context) error {
	log.Info("Starting MutatingAdmissionPolicy-based sidecar injection system")

	if !pm.enabled {
		log.Info("MutatingAdmissionPolicy injection is disabled, skipping startup")
		return nil
	}

	// Start the controller manager first
	log.Info("Starting controller manager")
	go func() {
		if err := pm.controller.mgr.Start(ctx); err != nil {
			log.Errorf("Failed to start controller manager: %v", err)
		}
	}()
	
	// Wait for cache to sync, then trigger reconciliation
	go func() {
		log.Info("Waiting for cache sync before reconciliation")
		if synced := pm.controller.mgr.GetCache().WaitForCacheSync(ctx); !synced {
			log.Error("Failed to sync cache")
			return
		}
		log.Info("Cache synced, triggering policy reconciliation")
		
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: pm.controller.namespace,
				Name:      "manual-reconcile",
			},
		}
		
		result, err := pm.controller.Reconcile(ctx, req)
		if err != nil {
			log.Errorf("Manual reconciliation failed: %v", err)
		} else {
			log.Infof("Manual reconciliation succeeded: %+v", result)
		}
	}()

	log.Info("MutatingAdmissionPolicy-based sidecar injection system started successfully")
	return nil
}

// Enable activates the policy-based injection system
func (pm *PolicyManager) Enable() {
	log.Info("Enabling MutatingAdmissionPolicy-based sidecar injection")
	pm.enabled = true
}

// Disable deactivates the policy-based injection system
func (pm *PolicyManager) Disable() {
	log.Info("Disabling MutatingAdmissionPolicy-based sidecar injection")
	pm.enabled = false
}

// IsEnabled returns whether the policy-based injection system is enabled
func (pm *PolicyManager) IsEnabled() bool {
	return pm.enabled
}

// GetController returns the underlying controller
func (pm *PolicyManager) GetController() *Controller {
	return pm.controller
}

// ValidateConfiguration validates the current configuration for policy-based injection
func (pm *PolicyManager) ValidateConfiguration() error {
	if pm.meshWatcher == nil {
		return fmt.Errorf("mesh watcher is not configured")
	}

	meshConfig := pm.meshWatcher.Mesh()
	if meshConfig == nil {
		return fmt.Errorf("mesh configuration is not available")
	}

	if pm.injectionConfig == nil {
		return fmt.Errorf("injection configuration is not available")
	}

	log.Info("MutatingAdmissionPolicy configuration validation successful")
	return nil
}

// GetStatus returns the current status of the policy-based injection system
func (pm *PolicyManager) GetStatus() map[string]any {
	status := map[string]any{
		"enabled":           pm.enabled,
		"controllerReady":   pm.controller != nil,
		"meshConfigLoaded":  pm.meshWatcher != nil && pm.meshWatcher.Mesh() != nil,
		"injectionConfigLoaded": pm.injectionConfig != nil,
	}

	if pm.controller != nil {
		status["currentRevision"] = pm.controller.currentRevision
		status["configMapName"] = pm.controller.configMapName
	}

	return status
}