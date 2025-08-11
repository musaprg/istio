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
	"testing"

	meshconfig "istio.io/api/mesh/v1alpha1"
	"istio.io/istio/pkg/config/mesh"
	assert "github.com/stretchr/testify/assert"
)

func TestTemplateFunctionConverter(t *testing.T) {
	meshConfig := mesh.DefaultMeshConfig()
	values := map[string]any{
		"global": map[string]any{
			"proxy": map[string]any{
				"image": "gcr.io/istio-testing/proxyv2:1.26.3",
			},
			"istioNamespace": "istio-system",
		},
	}
	
	converter := NewTemplateFunctionConverter(meshConfig, values, "default", "istio-system")

	t.Run("ConvertProtoToJSON", func(t *testing.T) {
		proxyConfig := &meshconfig.ProxyConfig{
			ConfigPath:    "/etc/istio/proxy",
			BinaryPath:    "/usr/local/bin/envoy",
			ClusterName:   &meshconfig.ProxyConfig_ServiceCluster{ServiceCluster: "test-cluster"},
		}

		jsonStr, err := converter.ConvertProtoToJSON(proxyConfig)
		assert.NoError(t, err)
		assert.NotEmpty(t, jsonStr)
		t.Logf("ProtoToJSON result: %s", jsonStr)
	})

	t.Run("ExcludeInboundPort", func(t *testing.T) {
		result := converter.ExcludeInboundPort("15020", "8080,9090")
		expected := "15090,15021,15020,8080,9090"
		assert.Equal(t, expected, result)
	})

	t.Run("AppendMultusNetwork", func(t *testing.T) {
		result := converter.AppendMultusNetwork("net1,net2", "default/istio-cni")
		expected := "net1,net2,default/istio-cni"
		assert.Equal(t, expected, result)
	})

	t.Run("AnnotationWithDefault", func(t *testing.T) {
		annotations := map[string]string{
			"key1": "value1",
		}
		
		// Test existing key
		result := converter.AnnotationWithDefault(annotations, "key1", "default")
		assert.Equal(t, "value1", result)
		
		// Test non-existing key
		result = converter.AnnotationWithDefault(annotations, "key2", "default")
		assert.Equal(t, "default", result)
	})

	t.Run("ProcessGlobalValues", func(t *testing.T) {
		processed := converter.ProcessGlobalValues()
		assert.NotEmpty(t, processed)
		assert.Equal(t, "gcr.io/istio-testing/proxyv2:1.26.3", processed["global.proxy.image"])
		assert.Equal(t, "istio-system", processed["global.istioNamespace"])
		t.Logf("Processed values: %v", processed)
	})
}

func TestInjectionParametersGeneration(t *testing.T) {
	// This is a placeholder for testing the full injection parameter generation
	// In a real implementation, this would require setting up a full controller
	// with mocked Kubernetes clients
	
	t.Run("BasicParameterGeneration", func(t *testing.T) {
		// Test that we can create the basic structures
		meshConfig := mesh.DefaultMeshConfig()
		
		converter := NewTemplateFunctionConverter(meshConfig, map[string]any{}, "default", "istio-system")
		assert.NotNil(t, converter)
		
		// Test basic functions work
		result := converter.ValueOrDefault("", "default-value")
		assert.Equal(t, "default-value", result)
		
		result = converter.ValueOrDefault("actual-value", "default-value")
		assert.Equal(t, "actual-value", result)
	})
}