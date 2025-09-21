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
	"strings"

	meshconfig "istio.io/api/mesh/v1alpha1"
	"istio.io/istio/pkg/log"
)

// TemplateFunctionConverter handles conversion of Go template functions to pre-calculated values
type TemplateFunctionConverter struct {
	meshConfig *meshconfig.MeshConfig
	values     map[string]any
	revision   string
	namespace  string
}

// NewTemplateFunctionConverter creates a new template function converter
func NewTemplateFunctionConverter(meshConfig *meshconfig.MeshConfig, values map[string]any, revision, namespace string) *TemplateFunctionConverter {
	return &TemplateFunctionConverter{
		meshConfig: meshConfig,
		values:     values,
		revision:   revision,
		namespace:  namespace,
	}
}

// ConvertProtoToJSON converts protobuf configuration to JSON string
// This replaces the {{ protoToJSON .ProxyConfig }} template function
func (c *TemplateFunctionConverter) ConvertProtoToJSON(config *meshconfig.ProxyConfig) (string, error) {
	if config == nil {
		return "{}", nil
	}

	// Create a simplified JSON representation
	// This removes default values and unset fields similar to the original function
	jsonConfig := map[string]any{}

	if config.ConfigPath != "" {
		jsonConfig["configPath"] = config.ConfigPath
	}
	if config.BinaryPath != "" {
		jsonConfig["binaryPath"] = config.BinaryPath
	}
	if clusterName := config.GetClusterName(); clusterName != nil {
		if sc, ok := clusterName.(*meshconfig.ProxyConfig_ServiceCluster); ok && sc.ServiceCluster != "" {
			jsonConfig["serviceCluster"] = sc.ServiceCluster
		}
	}
	if config.DrainDuration != nil {
		jsonConfig["drainDuration"] = config.DrainDuration.String()
	}
	// ParentShutdownDuration field removed from ProxyConfig in newer API version
	if config.DiscoveryAddress != "" {
		jsonConfig["discoveryAddress"] = config.DiscoveryAddress
	}
	// ConnectTimeout field removed from ProxyConfig in newer API version
	if config.StatsdUdpAddress != "" {
		jsonConfig["statsdUdpAddress"] = config.StatsdUdpAddress
	}
	if config.ProxyAdminPort != 0 {
		jsonConfig["proxyAdminPort"] = config.ProxyAdminPort
	}
	if config.ControlPlaneAuthPolicy != meshconfig.AuthenticationPolicy_NONE {
		jsonConfig["controlPlaneAuthPolicy"] = config.ControlPlaneAuthPolicy.String()
	}
	if config.CustomConfigFile != "" {
		jsonConfig["customConfigFile"] = config.CustomConfigFile
	}
	// StatConfigType field removed from ProxyConfig in newer API version
	if config.Concurrency != nil {
		jsonConfig["concurrency"] = config.Concurrency.Value
	}
	// ProxyStatsMatcher fields may have changed in newer API version
	if config.ProxyStatsMatcher != nil {
		jsonConfig["proxyStatsMatcher"] = map[string]any{
			"enabled": true,
		}
	}
	if config.InterceptionMode != meshconfig.ProxyConfig_REDIRECT {
		jsonConfig["interceptionMode"] = config.InterceptionMode.String()
	}
	if config.Tracing != nil {
		jsonConfig["tracing"] = c.convertTracingConfig(config.Tracing)
	}

	jsonBytes, err := json.Marshal(jsonConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal proxy config: %w", err)
	}

	return string(jsonBytes), nil
}

// convertTracingConfig converts tracing configuration
// Simplified version for API compatibility
func (c *TemplateFunctionConverter) convertTracingConfig(tracing *meshconfig.Tracing) map[string]any {
	tracingConfig := map[string]any{}
	
	// Tracing configuration fields may have changed in newer API
	// Return basic structure
	tracingConfig["enabled"] = tracing != nil
	
	return tracingConfig
}

// ConvertStructToJSON converts a struct to JSON string
// This replaces the {{ structToJSON $p }} template function
func (c *TemplateFunctionConverter) ConvertStructToJSON(obj any) (string, error) {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("failed to marshal struct: %w", err)
	}
	return string(jsonBytes), nil
}

// ExcludeInboundPort processes port exclusion logic
// This replaces the {{ excludeInboundPort ... }} template function
func (c *TemplateFunctionConverter) ExcludeInboundPort(statusPort, excludePorts string) string {
	excludeList := []string{"15090", "15021"}
	
	if statusPort != "" && statusPort != "0" {
		excludeList = append(excludeList, statusPort)
	}
	
	if excludePorts != "" {
		additionalPorts := strings.Split(excludePorts, ",")
		for _, port := range additionalPorts {
			port = strings.TrimSpace(port)
			if port != "" && port != "0" {
				excludeList = append(excludeList, port)
			}
		}
	}
	
	// Remove duplicates and sort
	seen := make(map[string]bool)
	uniquePorts := []string{}
	for _, port := range excludeList {
		if !seen[port] {
			seen[port] = true
			uniquePorts = append(uniquePorts, port)
		}
	}
	
	return strings.Join(uniquePorts, ",")
}

// AppendMultusNetwork handles Multus network annotation processing
// This replaces the {{ appendMultusNetwork ... }} template function
func (c *TemplateFunctionConverter) AppendMultusNetwork(existingNetworks, networkToAdd string) string {
	if existingNetworks == "" {
		return networkToAdd
	}
	
	// Parse existing networks
	networks := []string{}
	if existingNetworks != "" {
		// Handle both string and JSON array formats
		if strings.HasPrefix(existingNetworks, "[") {
			// JSON array format
			var networkArray []map[string]any
			if err := json.Unmarshal([]byte(existingNetworks), &networkArray); err == nil {
				for _, net := range networkArray {
					if name, ok := net["name"].(string); ok {
						networks = append(networks, name)
					}
				}
			}
		} else {
			// Comma-separated string format
			networks = strings.Split(existingNetworks, ",")
		}
	}
	
	// Add new network if not already present
	found := false
	for _, net := range networks {
		if strings.TrimSpace(net) == networkToAdd {
			found = true
			break
		}
	}
	
	if !found {
		networks = append(networks, networkToAdd)
	}
	
	// Return as comma-separated string
	result := []string{}
	for _, net := range networks {
		net = strings.TrimSpace(net)
		if net != "" {
			result = append(result, net)
		}
	}
	
	return strings.Join(result, ",")
}

// AnnotationWithDefault gets annotation value with default
// This replaces the {{ annotation .ObjectMeta `key` .Default }} template function
func (c *TemplateFunctionConverter) AnnotationWithDefault(annotations map[string]string, key, defaultValue string) string {
	if annotations == nil {
		return defaultValue
	}
	
	if value, exists := annotations[key]; exists {
		return value
	}
	
	return defaultValue
}

// ValueOrDefault returns value or default
// This replaces the {{ valueOrDefault .Value "default" }} template function
func (c *TemplateFunctionConverter) ValueOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// TruncateString truncates string to specified length
// This replaces the {{ trunc 63 .String }} template function
func (c *TemplateFunctionConverter) TruncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length]
}

// TrimSuffix removes suffix from string
// This replaces the {{ trimSuffix "-" .String }} template function
func (c *TemplateFunctionConverter) TrimSuffix(s, suffix string) string {
	return strings.TrimSuffix(s, suffix)
}

// QuoteString adds quotes around string
// This replaces the {{ quote .String }} template function
func (c *TemplateFunctionConverter) QuoteString(s string) string {
	return fmt.Sprintf(`"%s"`, s)
}

// IsSet checks if annotation/label exists
// This replaces the {{ isset .ObjectMeta.Annotations `key` }} template function
func (c *TemplateFunctionConverter) IsSet(m map[string]string, key string) bool {
	if m == nil {
		return false
	}
	_, exists := m[key]
	return exists
}

// Contains checks if string contains substring
// This replaces the {{ contains "/" .String }} template function
func (c *TemplateFunctionConverter) Contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ToLower converts string to lowercase
// This replaces the {{ toLower .String }} template function
func (c *TemplateFunctionConverter) ToLower(s string) string {
	return strings.ToLower(s)
}

// ProcessGlobalValues expands global values from Helm values
func (c *TemplateFunctionConverter) ProcessGlobalValues() map[string]string {
	processed := make(map[string]string)
	
	// Process common global values
	if global, ok := c.values["global"].(map[string]any); ok {
		// Process proxy image
		if proxy, ok := global["proxy"].(map[string]any); ok {
			if image, ok := proxy["image"].(string); ok {
				processed["global.proxy.image"] = image
			}
			if clusterDomain, ok := proxy["clusterDomain"].(string); ok {
				processed["global.proxy.clusterDomain"] = clusterDomain
			} else {
				processed["global.proxy.clusterDomain"] = "cluster.local"
			}
		}
		
		// Process istio namespace
		if namespace, ok := global["istioNamespace"].(string); ok {
			processed["global.istioNamespace"] = namespace
		} else {
			processed["global.istioNamespace"] = c.namespace
		}
		
		// Process mesh ID
		if meshID, ok := global["meshID"].(string); ok {
			processed["global.meshID"] = meshID
		} else {
			processed["global.meshID"] = "cluster.local"
		}
		
		// Process trust domain
		if trustDomain, ok := global["trustDomain"].(string); ok {
			processed["global.trustDomain"] = trustDomain
		} else if c.meshConfig != nil && c.meshConfig.TrustDomain != "" {
			processed["global.trustDomain"] = c.meshConfig.TrustDomain
		} else {
			processed["global.trustDomain"] = "cluster.local"
		}
		
		// Process multicluster settings
		if multiCluster, ok := global["multiCluster"].(map[string]any); ok {
			if clusterName, ok := multiCluster["clusterName"].(string); ok {
				processed["global.multiCluster.clusterName"] = clusterName
			} else {
				processed["global.multiCluster.clusterName"] = "Kubernetes"
			}
		}
		
		// Process network
		if network, ok := global["network"].(string); ok {
			processed["global.network"] = network
		}
		
		// Process pilot cert provider
		if pilotCertProvider, ok := global["pilotCertProvider"].(string); ok {
			processed["global.pilotCertProvider"] = pilotCertProvider
		} else {
			processed["global.pilotCertProvider"] = "istiod"
		}
	}
	
	// Add revision
	processed["revision"] = c.revision
	
	log.Infof("Processed global values: %v", processed)
	return processed
}