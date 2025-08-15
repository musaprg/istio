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

package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	admissionregistrationv1alpha1 "k8s.io/api/admissionregistration/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"istio.io/istio/pkg/env"
	"istio.io/istio/pkg/kube/inject"
	"istio.io/istio/pkg/kube/inject/policy"
	"istio.io/istio/pkg/log"
)

var (
	// Enable policy-based injection (MutatingAdmissionPolicy)
	policyInjectionEnabled = env.Register("POLICY_INJECTION_ENABLED", false, 
		"Enable MutatingAdmissionPolicy-based injection instead of webhook.")
	
	// Force policy-based injection even if webhook is enabled
	policyInjectionForce = env.Register("POLICY_INJECTION_FORCE", false,
		"Force MutatingAdmissionPolicy-based injection even if webhook injection is enabled.")
)

// initPolicyInjector initializes the MutatingAdmissionPolicy-based sidecar injector
func (s *Server) initPolicyInjector(args *PilotArgs) (*policy.PolicyManager, error) {
	log.Info("DEBUG: initPolicyInjector called")
	log.Infof("DEBUG: POLICY_INJECTION_ENABLED=%v", policyInjectionEnabled.Get())
	log.Infof("DEBUG: POLICY_INJECTION_FORCE=%v", policyInjectionForce.Get())
	
	if !policyInjectionEnabled.Get() {
		log.Info("MutatingAdmissionPolicy-based injection is disabled")
		return nil, nil
	}

	// Check if we should use policy-based injection
	if injectionEnabled.Get() && !policyInjectionForce.Get() {
		log.Info("Webhook injection is enabled, skipping policy-based injection. " +
			"Set POLICY_INJECTION_FORCE=true to use policy-based injection alongside webhook.")
		return nil, nil
	}

	// currently the constant: "./var/lib/istio/inject"
	injectPath := args.InjectionOptions.InjectionDirectory
	if injectPath == "" {
		log.Infof("Skipping policy injector, injection path is missing.")
		return nil, nil
	}

	// Load injection configuration
	var injectionConfig *inject.Config
	var valuesConfig string
	
	// Try to load from local files first
	if _, err := os.Stat(filepath.Join(injectPath, "config")); !os.IsNotExist(err) {
		configFile := filepath.Join(injectPath, "config")
		valuesFile := filepath.Join(injectPath, "values")
		
		configData, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		
		unmarshaledConfig, err := inject.UnmarshalConfig(configData)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
		injectionConfig = &unmarshaledConfig
		
		valuesData, err := os.ReadFile(valuesFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read values file: %w", err)
		}
		valuesConfig = string(valuesData)
		
	} else if s.kubeClient != nil {
		// Try to load from ConfigMap
		configMapName := getInjectorConfigMapName(args.Revision)
		cms := s.kubeClient.Kube().CoreV1().ConfigMaps(args.Namespace)
		cm, err := cms.Get(context.TODO(), configMapName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				log.Infof("Skipping policy injector, template ConfigMap not found")
				return nil, nil
			}
			return nil, err
		}
		
		configData, ok := cm.Data["config"]
		if !ok {
			return nil, fmt.Errorf("config key not found in ConfigMap %s", configMapName)
		}
		
		unmarshaledConfig2, err := inject.UnmarshalConfig([]byte(configData))
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal config from ConfigMap: %w", err)
		}
		injectionConfig = &unmarshaledConfig2
		
		valuesConfig, ok = cm.Data["values"]
		if !ok {
			log.Warn("values key not found in ConfigMap, using empty values")
			valuesConfig = "{}"
		}
	} else {
		log.Infof("Skipping policy injector, template not found")
		return nil, nil
	}

	log.Info("Initializing MutatingAdmissionPolicy-based sidecar injector")

	// Parse values into map for the controller
	values := make(map[string]any)
	vc, err := inject.NewValuesConfig(valuesConfig)
	if err != nil {
		log.Warnf("Failed to parse values config, using empty values: %v", err)
	} else {
		values = vc.Map()
	}

	// We need a controller-runtime manager for the policy controller
	// Check if we already have one or need to create one
	var mgr manager.Manager
	if s.controllerManager != nil {
		mgr = s.controllerManager
	} else {
		// Create a minimal manager for the policy controller
		mgr, err = s.createControllerManager(args)
		if err != nil {
			return nil, fmt.Errorf("failed to create controller manager: %w", err)
		}
		s.controllerManager = mgr
	}

	// Create the PolicyManager
	policyMgr, err := policy.NewPolicyManager(
		mgr,
		s.environment.Watcher,
		injectionConfig,
		args.Namespace,
		args.Revision,
		values,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create policy manager: %w", err)
	}

	// Enable the policy manager
	policyMgr.Enable()

	// Validate configuration
	if err := policyMgr.ValidateConfiguration(); err != nil {
		return nil, fmt.Errorf("policy configuration validation failed: %w", err)
	}

	// Add start function for the policy manager
	s.addStartFunc("policy injection controller", func(stop <-chan struct{}) error {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-stop
			cancel()
		}()
		
		// Start the policy manager
		if err := policyMgr.Start(ctx); err != nil {
			log.Errorf("Failed to start policy manager: %v", err)
			return err
		}
		
		log.Info("MutatingAdmissionPolicy-based sidecar injector started successfully")
		return nil
	})

	// If we created a new controller manager, start it
	if s.controllerManager != nil && mgr == s.controllerManager {
		s.addStartFunc("controller manager", func(stop <-chan struct{}) error {
			ctx, cancel := context.WithCancel(context.Background())
			go func() {
				<-stop
				cancel()
			}()
			
			go func() {
				if err := mgr.Start(ctx); err != nil {
					log.Errorf("Failed to start controller manager: %v", err)
				}
			}()
			return nil
		})
	}

	log.Info("MutatingAdmissionPolicy-based sidecar injector initialized")
	return policyMgr, nil
}

// createControllerManager creates a controller-runtime manager for the policy controller
func (s *Server) createControllerManager(args *PilotArgs) (manager.Manager, error) {
	if s.kubeClient == nil {
		return nil, fmt.Errorf("kubernetes client is required for controller manager")
	}

	// Get the rest config from the kube client
	restConfig := s.kubeClient.RESTConfig()

	// Create a scheme for the controller runtime
	// We need to add the types that the policy controller will manage
	scheme, err := createControllerScheme()
	if err != nil {
		return nil, fmt.Errorf("failed to create scheme: %w", err)
	}

	// Create manager options
	options := manager.Options{
		Scheme: scheme,
		Cache: cache.Options{
			// Watch all namespaces to include cluster-scoped resources
			// This is needed for MutatingAdmissionPolicy which is cluster-scoped
		},
		// We don't need leader election for the policy controller as each revision
		// manages its own policies
		LeaderElection: false,
		// Disable metrics server to avoid port conflict with istiod's :8080
		Metrics: server.Options{
			BindAddress: "0", // Disable metrics server
		},
		// Disable health probe server as well
		HealthProbeBindAddress: "0",
	}

	// Create the manager
	mgr, err := manager.New(restConfig, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create controller-runtime manager: %w", err)
	}

	return mgr, nil
}

// createControllerScheme creates the scheme for the controller runtime
func createControllerScheme() (*runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	
	// Add core types
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	
	// Add admission registration types (for MutatingAdmissionPolicy)
	if err := admissionregistrationv1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	
	return scheme, nil
}