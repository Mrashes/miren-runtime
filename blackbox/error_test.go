//go:build blackbox

package blackbox

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"miren.dev/runtime/blackbox/harness"
)

func TestCrashLoop(t *testing.T) {
	c := harness.NewCluster(t)
	m := harness.NewMiren(t, c)

	name := harness.UniqueAppName(t, "crash-loop")

	m.MustRun("deploy", "-a", name, "-d", m.ContainerPath(c.TestdataDir+"/crash-loop"), "-f")

	t.Cleanup(func() {
		m.Run("app", "delete", name, "-f")
	})

	// Wait for it to show up as crashed in app list
	harness.Poll(t, "app crashed", 2*time.Minute, 3*time.Second, func() (bool, string) {
		r := m.Run("app", "list", "--format", "json")
		if !r.Success() {
			return false, "app list failed"
		}

		var apps []struct {
			Name   string `json:"name"`
			Health string `json:"health"`
		}
		if err := json.Unmarshal([]byte(r.Stdout), &apps); err != nil {
			return false, fmt.Sprintf("parse error: %v", err)
		}

		for _, app := range apps {
			if app.Name == name {
				if app.Health == "crashed" {
					return true, ""
				}
				return false, fmt.Sprintf("health: %s", app.Health)
			}
		}
		return false, "app not found"
	})
}

func TestBadCommand(t *testing.T) {
	c := harness.NewCluster(t)
	m := harness.NewMiren(t, c)

	name := harness.UniqueAppName(t, "bad-cmd")

	m.MustRun("deploy", "-a", name, "-d", m.ContainerPath(c.TestdataDir+"/bad-command"), "-f")

	t.Cleanup(func() {
		m.Run("app", "delete", name, "-f")
	})

	// App with a bad command should eventually show as crashed
	harness.Poll(t, "app crashed", 2*time.Minute, 3*time.Second, func() (bool, string) {
		r := m.Run("app", "list", "--format", "json")
		if !r.Success() {
			return false, "app list failed"
		}

		var apps []struct {
			Name   string `json:"name"`
			Health string `json:"health"`
		}
		if err := json.Unmarshal([]byte(r.Stdout), &apps); err != nil {
			return false, fmt.Sprintf("parse error: %v", err)
		}

		for _, app := range apps {
			if app.Name == name {
				if app.Health == "crashed" {
					return true, ""
				}
				return false, fmt.Sprintf("health: %s", app.Health)
			}
		}
		return false, "app not found"
	})
}
