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
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	meshconfig "istio.io/api/mesh/v1alpha1"
	"istio.io/istio/pkg/log"
	_ "istio.io/istio/pkg/kube/inject" // unused for now
)

// InjectionParameters contains all pre-calculated injection configuration
type InjectionParameters struct {
	SidecarContainer     corev1.Container                  `json:"sidecarContainer"`
	InitContainer        corev1.Container                  `json:"initContainer"`
	BaseVolumes          []corev1.Volume                   `json:"baseVolumes"`
	CalculatedAnnotations map[string]string                `json:"calculatedAnnotations"`
	StaticEnvVars        []corev1.EnvVar                   `json:"staticEnvVars"`
	ProxyConfig          map[string]any                    `json:"proxyConfig"`
}

// generateInjectionConfigMap creates a ConfigMap with pre-calculated injection parameters
func (c *Controller) generateInjectionConfigMap() (*corev1.ConfigMap, error) {
	log.Info("Generating injection ConfigMap with pre-calculated parameters")

	meshConfig := c.meshWatcher.Mesh()
	if meshConfig == nil {
		return nil, fmt.Errorf("mesh configuration not available")
	}

	// Generate injection parameters
	params, err := c.generateInjectionParameters(meshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to generate injection parameters: %w", err)
	}

	// Convert to JSON strings for ConfigMap storage
	sidecarJSON, err := json.MarshalIndent(params.SidecarContainer, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sidecar container: %w", err)
	}

	initJSON, err := json.MarshalIndent(params.InitContainer, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal init container: %w", err)
	}

	volumesJSON, err := json.MarshalIndent(params.BaseVolumes, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal base volumes: %w", err)
	}

	annotationsJSON, err := json.MarshalIndent(params.CalculatedAnnotations, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal calculated annotations: %w", err)
	}

	staticEnvJSON, err := json.MarshalIndent(params.StaticEnvVars, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal static env vars: %w", err)
	}

	proxyConfigJSON, err := json.MarshalIndent(params.ProxyConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal proxy config: %w", err)
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.configMapName,
			Namespace: c.namespace,
			Labels: map[string]string{
				"istio.io/config-type": "injection-parameters",
				"istio.io/revision":    c.revision,
				"app.kubernetes.io/managed-by": PolicyControllerName,
			},
		},
		Data: map[string]string{
			"sidecar-container.json":       string(sidecarJSON),
			"init-container.json":          string(initJSON),
			"base-volumes.json":            string(volumesJSON),
			"calculated-annotations.json":  string(annotationsJSON),
			"static-env-vars.json":         string(staticEnvJSON),
			"proxy-config.json":            string(proxyConfigJSON),
		},
	}

	return configMap, nil
}

// generateInjectionParameters creates pre-calculated injection parameters
func (c *Controller) generateInjectionParameters(meshConfig *meshconfig.MeshConfig) (*InjectionParameters, error) {
	// Get proxy image
	proxyImage := c.getProxyImage()

	// Generate sidecar container
	sidecarContainer := c.generateSidecarContainer(meshConfig, proxyImage)

	// Generate init container
	initContainer := c.generateInitContainer(meshConfig, proxyImage)

	// Generate base volumes
	baseVolumes := c.generateBaseVolumes(meshConfig)

	// Generate calculated annotations
	calculatedAnnotations := c.generateCalculatedAnnotations(meshConfig)

	// Generate static environment variables
	staticEnvVars := c.generateStaticEnvVars(meshConfig)

	// Generate proxy configuration
	proxyConfig := c.generateProxyConfig(meshConfig)

	return &InjectionParameters{
		SidecarContainer:      sidecarContainer,
		InitContainer:         initContainer,
		BaseVolumes:           baseVolumes,
		CalculatedAnnotations: calculatedAnnotations,
		StaticEnvVars:         staticEnvVars,
		ProxyConfig:           proxyConfig,
	}, nil
}

// generateSidecarContainer creates the istio-proxy sidecar container specification
func (c *Controller) generateSidecarContainer(meshConfig *meshconfig.MeshConfig, proxyImage string) corev1.Container {
	return corev1.Container{
		Name:  "istio-proxy",
		Image: proxyImage,
		Args: []string{
			"proxy",
			"sidecar",
			"--domain",
			"$(POD_NAMESPACE).svc.cluster.local",
			"--proxyLogLevel=warning",
			"--proxyComponentLogLevel=misc:error",
			"--log_output_level=default:info",
		},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 15090,
				Protocol:      corev1.ProtocolTCP,
				Name:          "http-envoy-prom",
			},
		},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: &[]bool{false}[0],
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
			Privileged:               &[]bool{false}[0],
			ReadOnlyRootFilesystem:   &[]bool{true}[0],
			RunAsNonRoot:             &[]bool{true}[0],
			RunAsUser:                &[]int64{1337}[0],
			RunAsGroup:               &[]int64{1337}[0],
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    parseQuantity("100m"),
				corev1.ResourceMemory: parseQuantity("128Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    parseQuantity("2000m"),
				corev1.ResourceMemory: parseQuantity("1Gi"),
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "workload-socket", MountPath: "/var/run/secrets/workload-spiffe-uds"},
			{Name: "credential-socket", MountPath: "/var/run/secrets/credential-uds"},
			{Name: "workload-certs", MountPath: "/var/run/secrets/workload-spiffe-credentials"},
			{Name: "istiod-ca-cert", MountPath: "/var/run/secrets/istio"},
			{Name: "istio-ca-crl", MountPath: "/var/run/secrets/istio/crl"},
			{Name: "istio-data", MountPath: "/var/lib/istio/data"},
			{Name: "istio-envoy", MountPath: "/etc/istio/proxy"},
			{Name: "istio-token", MountPath: "/var/run/secrets/tokens"},
			{Name: "istio-podinfo", MountPath: "/etc/istio/pod"},
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/healthz/ready",
					Port: intstr.FromInt(15021),
				},
			},
			InitialDelaySeconds: 0,
			PeriodSeconds:       15,
			TimeoutSeconds:      3,
			FailureThreshold:    4,
		},
		// Note: Env vars will be added dynamically by CEL expressions
	}
}

// generateInitContainer creates the istio-init container specification
func (c *Controller) generateInitContainer(meshConfig *meshconfig.MeshConfig, proxyImage string) corev1.Container {
	proxyListenPort := "15001"
	if meshConfig.ProxyListenPort != 0 {
		proxyListenPort = fmt.Sprintf("%d", meshConfig.ProxyListenPort)
	}
	
	proxyInboundPort := "15006"
	if meshConfig.ProxyInboundListenPort != 0 {
		proxyInboundPort = fmt.Sprintf("%d", meshConfig.ProxyInboundListenPort)
	}

	return corev1.Container{
		Name:  "istio-init",
		Image: proxyImage,
		Args: []string{
			"istio-iptables",
			"-p", proxyListenPort,
			"-z", proxyInboundPort,
			"-u", "1337",
			"-m", "REDIRECT",
			"-i", "*",
			"-x", "",
			"-b", "*",
			"-d", "15090,15021,15020",
			"--log_output_level=default:info",
		},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: &[]bool{false}[0],
			Capabilities: &corev1.Capabilities{
				Add:  []corev1.Capability{"NET_ADMIN", "NET_RAW"},
				Drop: []corev1.Capability{"ALL"},
			},
			Privileged:               &[]bool{false}[0],
			ReadOnlyRootFilesystem:   &[]bool{false}[0],
			RunAsGroup:               &[]int64{0}[0],
			RunAsNonRoot:             &[]bool{false}[0],
			RunAsUser:                &[]int64{0}[0],
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    parseQuantity("100m"),
				corev1.ResourceMemory: parseQuantity("128Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    parseQuantity("2000m"),
				corev1.ResourceMemory: parseQuantity("1Gi"),
			},
		},
	}
}

// generateBaseVolumes creates the base volume specifications
func (c *Controller) generateBaseVolumes(meshConfig *meshconfig.MeshConfig) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "workload-socket",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "credential-socket",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "workload-certs",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "istio-envoy",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		},
		{
			Name: "istio-data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "istio-podinfo",
			VolumeSource: corev1.VolumeSource{
				DownwardAPI: &corev1.DownwardAPIVolumeSource{
					Items: []corev1.DownwardAPIVolumeFile{
						{
							Path: "labels",
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "metadata.labels",
							},
						},
						{
							Path: "annotations",
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "metadata.annotations",
							},
						},
					},
				},
			},
		},
		{
			Name: "istio-token",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: []corev1.VolumeProjection{
						{
							ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
								Path:              "istio-token",
								ExpirationSeconds: &[]int64{43200}[0],
								Audience:          "istio-ca",
							},
						},
					},
				},
			},
		},
		{
			Name: "istiod-ca-cert",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "istio-ca-root-cert",
					},
				},
			},
		},
		{
			Name: "istio-ca-crl",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "istio-ca-crl",
					},
					Optional: &[]bool{true}[0],
				},
			},
		},
	}
}

// generateCalculatedAnnotations creates pre-calculated annotations
func (c *Controller) generateCalculatedAnnotations(meshConfig *meshconfig.MeshConfig) map[string]string {
	annotations := map[string]string{
		"istio.io/rev":                                  c.revision,
		"sidecar.istio.io/interceptionMode":            "REDIRECT",
		"traffic.sidecar.istio.io/includeInboundPorts": "*",
		"traffic.sidecar.istio.io/excludeInboundPorts": "15090,15021,15020",
	}

	// Note: Mesh-specific annotations will be handled in CEL expressions
	// based on ProxyConfig.InterceptionMode, not MeshConfig

	return annotations
}

// generateStaticEnvVars creates static environment variables
func (c *Controller) generateStaticEnvVars(meshConfig *meshconfig.MeshConfig) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{
			Name:  "PILOT_CERT_PROVIDER",
			Value: "istiod",
		},
		{
			Name:  "CA_ADDR",
			Value: fmt.Sprintf("istiod.%s.svc:15012", c.namespace),
		},
		{
			Name:  "ISTIO_META_CLUSTER_ID",
			Value: "Kubernetes",
		},
		{
			Name:  "ISTIO_META_INTERCEPTION_MODE",
			Value: "REDIRECT",
		},
		{
			Name:  "ISTIO_META_MESH_ID",
			Value: "cluster.local",
		},
		{
			Name:  "TRUST_DOMAIN",
			Value: "cluster.local",
		},
	}

	// Add mesh-specific environment variables
	if meshConfig.TrustDomain != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name:  "TRUST_DOMAIN",
			Value: meshConfig.TrustDomain,
		}, corev1.EnvVar{
			Name:  "ISTIO_META_MESH_ID",
			Value: meshConfig.TrustDomain,
		})
	}

	return envVars
}

// generateProxyConfig creates proxy configuration for templates
func (c *Controller) generateProxyConfig(meshConfig *meshconfig.MeshConfig) map[string]any {
	// Convert protobuf to JSON-serializable format
	// This replaces the complex protoToJSON template function
	return map[string]any{
		"concurrency":         0,
		"tracing":            map[string]any{},
		"interceptionMode":   "REDIRECT",
		"holdApplicationUntilProxyStarts": false,
	}
}

// getProxyImage returns the proxy image to use
func (c *Controller) getProxyImage() string {
	if image, ok := c.values["global.proxy.image"].(string); ok && image != "" {
		return image
	}
	return "gcr.io/istio-testing/proxyv2:latest" // Default image
}

// Helper functions

func parseQuantity(s string) resource.Quantity {
	q, _ := resource.ParseQuantity(s)
	return q
}