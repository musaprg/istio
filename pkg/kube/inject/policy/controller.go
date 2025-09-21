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
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"istio.io/istio/pkg/config/mesh"
	"istio.io/istio/pkg/kube/inject"
	"istio.io/istio/pkg/log"
)

const (
	// PolicyControllerName is the name of the MutatingAdmissionPolicy controller
	PolicyControllerName = "istio-mutating-admission-policy-controller"
	
	// ConfigMapNamePrefix is the prefix for injection configuration ConfigMaps
	ConfigMapNamePrefix = "istio-injection-config-rev"
	
	// PolicyNamePrefix is the prefix for MutatingAdmissionPolicy resources
	PolicyNamePrefix = "istio-sidecar-injection"
	
	// FinalizerName is the finalizer added to managed resources
	FinalizerName = "policy.injection.istio.io/finalizer"
)

// Controller manages MutatingAdmissionPolicy resources for Istio sidecar injection
type Controller struct {
	client.Client
	mgr            manager.Manager
	meshWatcher    mesh.Watcher
	injectionConfig *inject.Config
	queue          workqueue.TypedRateLimitingInterface[reconcile.Request]
	
	// Configuration
	namespace      string
	revision       string
	values         map[string]any
	
	// State tracking
	currentRevision int64
	configMapName   string
}

// NewController creates a new MutatingAdmissionPolicy controller
func NewController(mgr manager.Manager, meshWatcher mesh.Watcher, injectionConfig *inject.Config, namespace, revision string, values map[string]any) (*Controller, error) {
	c := &Controller{
		Client:          mgr.GetClient(),
		mgr:             mgr,
		meshWatcher:     meshWatcher,
		injectionConfig: injectionConfig,
		namespace:       namespace,
		revision:        revision,
		values:          values,
		queue:          workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]()),
		currentRevision: time.Now().Unix(),
	}

	// For now, we'll use manual triggering via mesh config changes
	// In production, we'd set up proper controller watches
	log.Info("Policy injection controller initialized")

	// Watch mesh configuration changes
	meshWatcher.AddMeshHandler(func() {
		log.Infof("Mesh configuration changed, triggering policy reconciliation")
		c.queue.Add(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: c.namespace,
				Name:      "mesh-config-change",
			},
		})
	})
	
	// Trigger initial reconciliation
	log.Info("Triggering initial policy reconciliation")
	c.queue.Add(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: c.namespace,
			Name:      "initial-reconcile",
		},
	})

	return c, nil
}

// Reconcile handles reconciliation of MutatingAdmissionPolicy resources (DEPRECATED - use ReconcileUnstructured)
func (c *Controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log.Info("DEPRECATED: Reconcile called - this method is deprecated, use ReconcileUnstructured instead")
	return reconcile.Result{}, nil
}

// Start starts the controller
func (c *Controller) Start(ctx context.Context) error {
	log.Info("Starting MutatingAdmissionPolicy controller")
	
	// Perform initial reconciliation
	c.queue.Add(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: c.namespace,
			Name:      "initial-reconciliation",
		},
	})

	return nil
}

// createOrUpdateConfigMap creates or updates the injection configuration ConfigMap
func (c *Controller) createOrUpdateConfigMap(ctx context.Context, configMap *corev1.ConfigMap) error {
	existing := &corev1.ConfigMap{}
	err := c.Get(ctx, types.NamespacedName{
		Namespace: configMap.Namespace,
		Name:      configMap.Name,
	}, existing)

	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to get existing ConfigMap: %w", err)
		}
		// ConfigMap doesn't exist, create it
		log.Infof("Creating ConfigMap %s/%s", configMap.Namespace, configMap.Name)
		return c.Create(ctx, configMap)
	}

	// ConfigMap exists, update it
	log.Infof("Updating ConfigMap %s/%s", configMap.Namespace, configMap.Name)
	existing.Data = configMap.Data
	existing.Labels = configMap.Labels
	return c.Update(ctx, existing)
}

// createOrUpdatePolicy creates or updates a MutatingAdmissionPolicy (DEPRECATED - using unstructured API instead)
func (c *Controller) createOrUpdatePolicy(ctx context.Context, policy interface{}) error {
	log.Info("DEPRECATED: createOrUpdatePolicy called - this method is deprecated")
	return nil
}

// createOrUpdatePolicyBinding creates or updates a MutatingAdmissionPolicyBinding (DEPRECATED - using unstructured API instead)
func (c *Controller) createOrUpdatePolicyBinding(ctx context.Context, binding interface{}) error {
	log.Info("DEPRECATED: createOrUpdatePolicyBinding called - this method is deprecated")
	return nil
}