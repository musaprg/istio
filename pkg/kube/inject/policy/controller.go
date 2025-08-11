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

	admissionregistrationv1alpha1 "k8s.io/api/admissionregistration/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
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

	// TODO: Set up controller with manager and watchers
	// For now, skip controller setup to get basic compilation working
	log.Info("Controller setup skipped for initial implementation")
	
	_ = controller.New // Silence unused import

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

	return c, nil
}

// Reconcile handles reconciliation of MutatingAdmissionPolicy resources
func (c *Controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log.Infof("Reconciling MutatingAdmissionPolicy for %s/%s", req.Namespace, req.Name)

	// Generate new revision number
	newRevision := time.Now().Unix()
	if newRevision == c.currentRevision {
		newRevision = c.currentRevision + 1
	}
	c.currentRevision = newRevision
	c.configMapName = fmt.Sprintf("%s-%d", ConfigMapNamePrefix, c.currentRevision)

	// Step 1: Generate injection configuration ConfigMap
	configMap, err := c.generateInjectionConfigMap()
	if err != nil {
		log.Errorf("Failed to generate injection ConfigMap: %v", err)
		return reconcile.Result{}, err
	}

	// Step 2: Create or update ConfigMap
	err = c.createOrUpdateConfigMap(ctx, configMap)
	if err != nil {
		log.Errorf("Failed to create/update ConfigMap: %v", err)
		return reconcile.Result{}, err
	}

	// Step 3: Generate MutatingAdmissionPolicy resources
	policies, err := c.generatePolicies()
	if err != nil {
		log.Errorf("Failed to generate policies: %v", err)
		return reconcile.Result{}, err
	}

	// Step 4: Create or update policies
	for _, policy := range policies {
		err = c.createOrUpdatePolicy(ctx, policy)
		if err != nil {
			log.Errorf("Failed to create/update policy %s: %v", policy.Name, err)
			return reconcile.Result{}, err
		}
	}

	// Step 5: Create or update policy bindings
	bindings, err := c.generatePolicyBindings()
	if err != nil {
		log.Errorf("Failed to generate policy bindings: %v", err)
		return reconcile.Result{}, err
	}

	for _, binding := range bindings {
		err = c.createOrUpdatePolicyBinding(ctx, binding)
		if err != nil {
			log.Errorf("Failed to create/update policy binding %s: %v", binding.Name, err)
			return reconcile.Result{}, err
		}
	}

	log.Infof("Successfully reconciled MutatingAdmissionPolicy resources with revision %d", c.currentRevision)
	return reconcile.Result{RequeueAfter: time.Minute * 5}, nil
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

// createOrUpdatePolicy creates or updates a MutatingAdmissionPolicy
func (c *Controller) createOrUpdatePolicy(ctx context.Context, policy *admissionregistrationv1alpha1.MutatingAdmissionPolicy) error {
	existing := &admissionregistrationv1alpha1.MutatingAdmissionPolicy{}
	err := c.Get(ctx, types.NamespacedName{Name: policy.Name}, existing)

	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to get existing policy: %w", err)
		}
		// Policy doesn't exist, create it
		log.Infof("Creating MutatingAdmissionPolicy %s", policy.Name)
		return c.Create(ctx, policy)
	}

	// Policy exists, update it
	log.Infof("Updating MutatingAdmissionPolicy %s", policy.Name)
	existing.Spec = policy.Spec
	existing.Labels = policy.Labels
	return c.Update(ctx, existing)
}

// createOrUpdatePolicyBinding creates or updates a MutatingAdmissionPolicyBinding
func (c *Controller) createOrUpdatePolicyBinding(ctx context.Context, binding *admissionregistrationv1alpha1.MutatingAdmissionPolicyBinding) error {
	existing := &admissionregistrationv1alpha1.MutatingAdmissionPolicyBinding{}
	err := c.Get(ctx, types.NamespacedName{Name: binding.Name}, existing)

	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to get existing policy binding: %w", err)
		}
		// Binding doesn't exist, create it
		log.Infof("Creating MutatingAdmissionPolicyBinding %s", binding.Name)
		return c.Create(ctx, binding)
	}

	// Binding exists, update it
	log.Infof("Updating MutatingAdmissionPolicyBinding %s", binding.Name)
	existing.Spec = binding.Spec
	existing.Labels = binding.Labels
	return c.Update(ctx, existing)
}