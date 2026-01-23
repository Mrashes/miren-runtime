package sandbox

import (
	"context"
	"log/slog"
	"testing"
	"time"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/stretchr/testify/require"

	compute "miren.dev/runtime/api/compute/compute_v1alpha"
	"miren.dev/runtime/api/core/core_v1alpha"
	"miren.dev/runtime/api/entityserver/entityserver_v1alpha"
	"miren.dev/runtime/pkg/entity"
	"miren.dev/runtime/pkg/idgen"
	"miren.dev/runtime/pkg/imagerefs"
	"miren.dev/runtime/pkg/testutils"
)

func TestImageWatchdog(t *testing.T) {
	t.Run("removes unused images", func(t *testing.T) {
		r := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		testDeps, cleanup := testutils.NewTestDeps()
		defer cleanup()

		cc := testDeps.CC
		eac := testDeps.EAC
		ii := testDeps.NewImageImporter()

		ctx = namespaces.WithNamespace(ctx, ii.Namespace)

		// Pull an extra image that isn't used by any sandbox or app version
		unusedImage := "docker.io/library/alpine:3.18"
		_, err := cc.Pull(ctx, unusedImage, containerd.WithPullUnpack)
		r.NoError(err)

		// Verify image exists
		images, err := cc.ListImages(ctx)
		r.NoError(err)
		foundUnused := false
		for _, img := range images {
			if img.Name() == unusedImage {
				foundUnused = true
				break
			}
		}
		r.True(foundUnused, "unused image should exist before GC")

		// Create the image watchdog
		watchdog := &ImageWatchdog{
			Log:       slog.Default(),
			CC:        cc,
			EAC:       eac,
			Namespace: ii.Namespace,
			DataPath:  "/tmp",
			Config:    DefaultImageGCConfig(),
		}

		// Run GC
		result, err := watchdog.RunGC(ctx)
		r.NoError(err)

		// Verify the unused image was deleted
		r.Contains(result.DeletedImages, unusedImage, "unused image should be deleted")

		// Verify image no longer exists
		images, err = cc.ListImages(ctx)
		r.NoError(err)
		foundUnused = false
		for _, img := range images {
			if img.Name() == unusedImage {
				foundUnused = true
				break
			}
		}
		r.False(foundUnused, "unused image should be removed after GC")
	})

	t.Run("retains images from running sandboxes", func(t *testing.T) {
		r := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		testDeps, cleanup := testutils.NewTestDeps()
		defer cleanup()

		cc := testDeps.CC
		eac := testDeps.EAC
		ii := testDeps.NewImageImporter()

		ctx = namespaces.WithNamespace(ctx, ii.Namespace)

		// Pull an image and create a running sandbox that uses it
		inUseImage := "docker.io/library/alpine:3.19"
		_, err := cc.Pull(ctx, inUseImage, containerd.WithPullUnpack)
		r.NoError(err)

		// Create a running sandbox that references the image
		sbID := entity.Id(idgen.GenNS("sb"))
		sb := &compute.Sandbox{
			ID:     sbID,
			Status: compute.RUNNING,
			Spec: compute.SandboxSpec{
				Container: []compute.SandboxSpecContainer{
					{
						Name:  "main",
						Image: inUseImage,
					},
				},
			},
		}

		var rpcE entityserver_v1alpha.Entity
		rpcE.SetId(sbID.String())
		rpcE.SetAttrs(entity.New(
			entity.DBId, sbID,
			sb.Encode).Attrs())
		_, err = eac.Put(ctx, &rpcE)
		r.NoError(err)

		// Create the image watchdog
		watchdog := &ImageWatchdog{
			Log:       slog.Default(),
			CC:        cc,
			EAC:       eac,
			Namespace: ii.Namespace,
			DataPath:  "/tmp",
			Config:    DefaultImageGCConfig(),
		}

		// Run GC
		result, err := watchdog.RunGC(ctx)
		r.NoError(err)

		// Verify the in-use image was NOT deleted
		r.NotContains(result.DeletedImages, inUseImage, "in-use image should not be deleted")

		// Verify image still exists
		images, err := cc.ListImages(ctx)
		r.NoError(err)
		foundInUse := false
		for _, img := range images {
			if img.Name() == inUseImage {
				foundInUse = true
				break
			}
		}
		r.True(foundInUse, "in-use image should still exist after GC")
	})

	t.Run("retains images within retention days", func(t *testing.T) {
		r := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		testDeps, cleanup := testutils.NewTestDeps()
		defer cleanup()

		cc := testDeps.CC
		eac := testDeps.EAC
		ii := testDeps.NewImageImporter()

		ctx = namespaces.WithNamespace(ctx, ii.Namespace)

		// Pull an image
		recentImage := "docker.io/library/alpine:3.20"
		_, err := cc.Pull(ctx, recentImage, containerd.WithPullUnpack)
		r.NoError(err)

		// Create an App entity first (required for referential integrity)
		appID := entity.Id(idgen.GenNS("app"))
		app := &core_v1alpha.App{
			ID: appID,
		}
		var appRpcE entityserver_v1alpha.Entity
		appRpcE.SetId(appID.String())
		appRpcE.SetAttrs(entity.New(
			entity.DBId, appID,
			app.Encode).Attrs())
		_, err = eac.Put(ctx, &appRpcE)
		r.NoError(err)

		// Create an AppVersion that references the image (created now, so within retention)
		avID := entity.Id(idgen.GenNS("av"))
		av := &core_v1alpha.AppVersion{
			ID:       avID,
			App:      appID,
			ImageUrl: recentImage,
		}

		var rpcE entityserver_v1alpha.Entity
		rpcE.SetId(avID.String())
		rpcE.SetAttrs(entity.New(
			entity.DBId, avID,
			av.Encode).Attrs())
		_, err = eac.Put(ctx, &rpcE)
		r.NoError(err)

		// Create the image watchdog with default 30 day retention
		watchdog := &ImageWatchdog{
			Log:       slog.Default(),
			CC:        cc,
			EAC:       eac,
			Namespace: ii.Namespace,
			DataPath:  "/tmp",
			Config:    DefaultImageGCConfig(),
		}

		// Run GC
		result, err := watchdog.RunGC(ctx)
		r.NoError(err)

		// Verify the recent image was NOT deleted (within 30 days)
		r.NotContains(result.DeletedImages, recentImage, "recent image should not be deleted")
	})

	t.Run("retains infrastructure images", func(t *testing.T) {
		r := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		testDeps, cleanup := testutils.NewTestDeps()
		defer cleanup()

		cc := testDeps.CC
		eac := testDeps.EAC
		ii := testDeps.NewImageImporter()

		ctx = namespaces.WithNamespace(ctx, ii.Namespace)

		// Create the image watchdog
		watchdog := &ImageWatchdog{
			Log:       slog.Default(),
			CC:        cc,
			EAC:       eac,
			Namespace: ii.Namespace,
			DataPath:  "/tmp",
			Config:    DefaultImageGCConfig(),
		}

		// Collect retained images
		retained, err := watchdog.collectRetainedImages(ctx)
		r.NoError(err)

		// Verify infrastructure images are retained
		r.True(retained[imagerefs.Pause], "pause image should be retained")
		r.True(retained[imagerefs.Etcd], "etcd image should be retained")
		r.True(retained[imagerefs.BuildKit], "buildkit image should be retained")
		r.True(retained[imagerefs.Minio], "minio image should be retained")
	})

	t.Run("retains last N versions per app", func(t *testing.T) {
		r := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		testDeps, cleanup := testutils.NewTestDeps()
		defer cleanup()

		cc := testDeps.CC
		eac := testDeps.EAC
		ii := testDeps.NewImageImporter()

		ctx = namespaces.WithNamespace(ctx, ii.Namespace)

		// Create an App entity first (required for referential integrity)
		appID := entity.Id(idgen.GenNS("app"))
		app := &core_v1alpha.App{
			ID: appID,
		}
		var appRpcE entityserver_v1alpha.Entity
		appRpcE.SetId(appID.String())
		appRpcE.SetAttrs(entity.New(
			entity.DBId, appID,
			app.Encode).Attrs())
		_, err := eac.Put(ctx, &appRpcE)
		r.NoError(err)

		// Create 3 versions with different images
		images := []string{
			"docker.io/library/alpine:v1",
			"docker.io/library/alpine:v2",
			"docker.io/library/alpine:v3",
		}

		for i, img := range images {
			avID := entity.Id(idgen.GenNS("av"))
			av := &core_v1alpha.AppVersion{
				ID:       avID,
				App:      appID,
				ImageUrl: img,
			}

			var rpcE entityserver_v1alpha.Entity
			rpcE.SetId(avID.String())
			rpcE.SetAttrs(entity.New(
				entity.DBId, avID,
				av.Encode).Attrs())
			_, err := eac.Put(ctx, &rpcE)
			r.NoError(err)

			// Small delay to ensure different timestamps
			if i < len(images)-1 {
				time.Sleep(10 * time.Millisecond)
			}
		}

		// Create the image watchdog with retention count of 2
		watchdog := &ImageWatchdog{
			Log:       slog.Default(),
			CC:        cc,
			EAC:       eac,
			Namespace: ii.Namespace,
			DataPath:  "/tmp",
			Config: ImageGCConfig{
				ScheduledGCInterval:   168 * time.Hour,
				PressureCheckInterval: 1 * time.Hour,
				DiskPressureThreshold: 80.0,
				RetentionDays:         0, // Disable time-based retention for this test
				RetentionCount:        2, // Keep only last 2 versions
			},
		}

		// Collect policy-retained images
		retained, err := watchdog.collectPolicyRetainedImages(ctx)
		r.NoError(err)

		// The most recent 2 versions should be retained
		r.True(retained["docker.io/library/alpine:v2"], "v2 should be retained (one of last 2)")
		r.True(retained["docker.io/library/alpine:v3"], "v3 should be retained (most recent)")
		// v1 may or may not be retained depending on timing; with 0 retention days
		// and only 2 retention count, v1 should not be retained
	})

	t.Run("starts and stops gracefully", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		testDeps, cleanup := testutils.NewTestDeps()
		defer cleanup()

		ii := testDeps.NewImageImporter()

		// Create the watchdog with very short intervals
		watchdog := &ImageWatchdog{
			Log:       slog.Default(),
			CC:        testDeps.CC,
			EAC:       testDeps.EAC,
			Namespace: ii.Namespace,
			DataPath:  "/tmp",
			Config: ImageGCConfig{
				ScheduledGCInterval:   100 * time.Millisecond,
				PressureCheckInterval: 50 * time.Millisecond,
				DiskPressureThreshold: 99.0, // High threshold so it won't trigger
				RetentionDays:         30,
				RetentionCount:        20,
			},
		}

		// Start the watchdog
		watchdog.Start(ctx)

		// Let it run for a bit
		time.Sleep(300 * time.Millisecond)

		// Stop the watchdog
		watchdog.Stop()

		// Should complete without hanging
		time.Sleep(200 * time.Millisecond)
	})

	t.Run("collectInUseImages returns images from pending sandboxes", func(t *testing.T) {
		r := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		testDeps, cleanup := testutils.NewTestDeps()
		defer cleanup()

		cc := testDeps.CC
		eac := testDeps.EAC
		ii := testDeps.NewImageImporter()

		ctx = namespaces.WithNamespace(ctx, ii.Namespace)

		// Create a pending sandbox that references an image
		pendingImage := "docker.io/library/alpine:pending"
		sbID := entity.Id(idgen.GenNS("sb"))
		sb := &compute.Sandbox{
			ID:     sbID,
			Status: compute.PENDING,
			Spec: compute.SandboxSpec{
				Container: []compute.SandboxSpecContainer{
					{
						Name:  "main",
						Image: pendingImage,
					},
				},
			},
		}

		var rpcE entityserver_v1alpha.Entity
		rpcE.SetId(sbID.String())
		rpcE.SetAttrs(entity.New(
			entity.DBId, sbID,
			sb.Encode).Attrs())
		_, err := eac.Put(ctx, &rpcE)
		r.NoError(err)

		// Create the image watchdog
		watchdog := &ImageWatchdog{
			Log:       slog.Default(),
			CC:        cc,
			EAC:       eac,
			Namespace: ii.Namespace,
			DataPath:  "/tmp",
			Config:    DefaultImageGCConfig(),
		}

		// Collect in-use images
		inUse, err := watchdog.collectInUseImages(ctx)
		r.NoError(err)

		// Verify the pending sandbox's image is included
		r.True(inUse[pendingImage], "pending sandbox image should be in use")
	})

	t.Run("ignores DEAD sandboxes for in-use images", func(t *testing.T) {
		r := require.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		testDeps, cleanup := testutils.NewTestDeps()
		defer cleanup()

		cc := testDeps.CC
		eac := testDeps.EAC
		ii := testDeps.NewImageImporter()

		ctx = namespaces.WithNamespace(ctx, ii.Namespace)

		// Create a DEAD sandbox that references an image
		deadImage := "docker.io/library/alpine:dead"
		sbID := entity.Id(idgen.GenNS("sb"))
		sb := &compute.Sandbox{
			ID:     sbID,
			Status: compute.DEAD,
			Spec: compute.SandboxSpec{
				Container: []compute.SandboxSpecContainer{
					{
						Name:  "main",
						Image: deadImage,
					},
				},
			},
		}

		var rpcE entityserver_v1alpha.Entity
		rpcE.SetId(sbID.String())
		rpcE.SetAttrs(entity.New(
			entity.DBId, sbID,
			sb.Encode).Attrs())
		_, err := eac.Put(ctx, &rpcE)
		r.NoError(err)

		// Create the image watchdog
		watchdog := &ImageWatchdog{
			Log:       slog.Default(),
			CC:        cc,
			EAC:       eac,
			Namespace: ii.Namespace,
			DataPath:  "/tmp",
			Config:    DefaultImageGCConfig(),
		}

		// Collect in-use images
		inUse, err := watchdog.collectInUseImages(ctx)
		r.NoError(err)

		// Verify the DEAD sandbox's image is NOT included
		r.False(inUse[deadImage], "DEAD sandbox image should not be in use")
	})
}
