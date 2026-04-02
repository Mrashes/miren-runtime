package valkey

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefinitionHasAllVariants(t *testing.T) {
	def := Definition()

	assert.Equal(t, AddonName, def.Name)
	assert.Equal(t, "Miren Valkey", def.DisplayName)
	assert.Equal(t, "small", def.DefaultVariant)
	assert.Len(t, def.Variants, 1)
	assert.Equal(t, "small", def.Variants[0].Name)
}

func TestBuildEnvVars(t *testing.T) {
	vars := buildEnvVars("myhost", 6379, "mypass")

	assert.Len(t, vars, 8)

	varMap := make(map[string]string)
	sensitiveMap := make(map[string]bool)
	for _, v := range vars {
		varMap[v.Key] = v.Value
		sensitiveMap[v.Key] = v.Sensitive
	}

	assert.Equal(t, "redis://:mypass@myhost:6379", varMap["VALKEY_URL"])
	assert.True(t, sensitiveMap["VALKEY_URL"])
	assert.Equal(t, "myhost", varMap["VALKEY_HOST"])
	assert.False(t, sensitiveMap["VALKEY_HOST"])
	assert.Equal(t, "6379", varMap["VALKEY_PORT"])
	assert.False(t, sensitiveMap["VALKEY_PORT"])
	assert.Equal(t, "mypass", varMap["VALKEY_PASSWORD"])
	assert.True(t, sensitiveMap["VALKEY_PASSWORD"])

	// REDIS_* aliases
	assert.Equal(t, varMap["VALKEY_URL"], varMap["REDIS_URL"])
	assert.True(t, sensitiveMap["REDIS_URL"])
	assert.Equal(t, "myhost", varMap["REDIS_HOST"])
	assert.Equal(t, "6379", varMap["REDIS_PORT"])
	assert.Equal(t, "mypass", varMap["REDIS_PASSWORD"])
	assert.True(t, sensitiveMap["REDIS_PASSWORD"])
}

func TestBuildValkeyURL(t *testing.T) {
	url := buildValkeyURL("host.example.com", 6379, "secret")
	assert.Equal(t, "redis://:secret@host.example.com:6379", url)
}

func TestVariantConfigContainsExpectedKeys(t *testing.T) {
	def := Definition()

	for _, variant := range def.Variants {
		t.Run(variant.Name, func(t *testing.T) {
			assert.NotEmpty(t, variant.Config[ConfigStorage])
		})
	}
}
