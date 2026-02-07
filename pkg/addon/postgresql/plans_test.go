package postgresql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefinitionHasAllPlans(t *testing.T) {
	def := Definition()

	assert.Equal(t, AddonName, def.Name)
	assert.Equal(t, "Miren PostgreSQL", def.DisplayName)
	assert.Equal(t, "small-local", def.DefaultPlan)
	assert.Len(t, def.Plans, 4)

	planNames := make(map[string]bool)
	for _, p := range def.Plans {
		planNames[p.Name] = true
	}

	assert.True(t, planNames["small-local"])
	assert.True(t, planNames["medium-local"])
	assert.True(t, planNames["large-local"])
	assert.True(t, planNames["shared"])
}

func TestIsSharedPlan(t *testing.T) {
	assert.True(t, IsSharedPlan("shared"))
	assert.False(t, IsSharedPlan("small-local"))
	assert.False(t, IsSharedPlan("medium-local"))
	assert.False(t, IsSharedPlan("large-local"))
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-app", "my_app"},
		{"MyApp", "myapp"},
		{"123app", "a123app"},
		{"app_name", "app_name"},
		{"app.name", "appname"},
		{"APP-NAME", "app_name"},
		{"", "app"},
		{"a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizeIdentifier(tt.input))
		})
	}
}

func TestBuildEnvVars(t *testing.T) {
	vars := buildEnvVars("myhost", 5432, "myuser", "mypass", "mydb")

	assert.Len(t, vars, 6)

	varMap := make(map[string]string)
	sensitiveMap := make(map[string]bool)
	for _, v := range vars {
		varMap[v.Key] = v.Value
		sensitiveMap[v.Key] = v.Sensitive
	}

	assert.Equal(t, "postgres://myuser:mypass@myhost:5432/mydb", varMap["DATABASE_URL"])
	assert.True(t, sensitiveMap["DATABASE_URL"])

	assert.Equal(t, "myhost", varMap["PGHOST"])
	assert.False(t, sensitiveMap["PGHOST"])

	assert.Equal(t, "5432", varMap["PGPORT"])
	assert.False(t, sensitiveMap["PGPORT"])

	assert.Equal(t, "myuser", varMap["PGUSER"])
	assert.False(t, sensitiveMap["PGUSER"])

	assert.Equal(t, "mypass", varMap["PGPASSWORD"])
	assert.True(t, sensitiveMap["PGPASSWORD"])

	assert.Equal(t, "mydb", varMap["PGDATABASE"])
	assert.False(t, sensitiveMap["PGDATABASE"])
}

func TestBuildDatabaseURL(t *testing.T) {
	url := buildDatabaseURL("host.example.com", 5432, "user", "pass", "dbname")
	assert.Equal(t, "postgres://user:pass@host.example.com:5432/dbname", url)
}

func TestPlanConfigContainsExpectedKeys(t *testing.T) {
	def := Definition()

	for _, plan := range def.Plans {
		t.Run(plan.Name, func(t *testing.T) {
			if plan.Name == "shared" {
				assert.Equal(t, "true", plan.Config[ConfigShared])
			} else {
				assert.NotEmpty(t, plan.Config[ConfigCPU])
				assert.NotEmpty(t, plan.Config[ConfigMemory])
				assert.NotEmpty(t, plan.Config[ConfigStorage])
				assert.Equal(t, "false", plan.Config[ConfigShared])
			}
		})
	}
}
