package sandbox

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/containerd/containerd/namespaces"
	containerd "github.com/containerd/containerd/v2/client"

	compute "miren.dev/runtime/api/compute/compute_v1alpha"
	"miren.dev/runtime/api/core/core_v1alpha"
	"miren.dev/runtime/api/entityserver/entityserver_v1alpha"
	"miren.dev/runtime/pkg/entity"
	"miren.dev/runtime/pkg/imagerefs"
	"miren.dev/runtime/pkg/sysstats"
)

// ImageGCConfig holds configuration for the image garbage collector.
type ImageGCConfig struct {
	// ScheduledGCInterval is how often to run scheduled GC regardless of pressure (default: 168h/weekly)
	ScheduledGCInterval time.Duration
	// PressureCheckInterval is how often to check disk pressure (default: 1h)
	PressureCheckInterval time.Duration
	// DiskPressureThreshold is the disk usage percentage that triggers immediate GC (default: 80%)
	DiskPressureThreshold float64
	// RetentionDays is how many days to keep images regardless of release count (default: 30)
	RetentionDays int
	// RetentionCount is how many recent releases per app to keep regardless of age (default: 20)
	RetentionCount int
}

// DefaultImageGCConfig returns the default configuration for image GC.
func DefaultImageGCConfig() ImageGCConfig {
	return ImageGCConfig{
		ScheduledGCInterval:   168 * time.Hour, // Weekly
		PressureCheckInterval: 1 * time.Hour,
		DiskPressureThreshold: 80.0,
		RetentionDays:         30,
		RetentionCount:        20,
	}
}

// ImageGCResult contains information about images cleaned up during GC.
type ImageGCResult struct {
	// DeletedImages contains names of images successfully removed
	DeletedImages []string
	// FailedImages contains names and errors for images that failed to be removed
	FailedImages map[string]error
	// TotalImages is the total number of images before GC
	TotalImages int
	// RetainedImages is the number of images kept
	RetainedImages int
}

// ImageWatchdog periodically garbage collects unused container images from containerd.
// It implements a retention policy that keeps images if they are:
// 1. Referenced by a running/pending sandbox
// 2. From an AppVersion less than RetentionDays old
// 3. Within the last RetentionCount AppVersions for their app
type ImageWatchdog struct {
	Log *slog.Logger
	CC  *containerd.Client
	EAC *entityserver_v1alpha.EntityAccessClient

	Namespace string
	DataPath  string
	Config    ImageGCConfig

	cancel context.CancelFunc
}

// Start begins the periodic image cleanup process.
func (w *ImageWatchdog) Start(ctx context.Context) {
	w.Log.Info("starting image watchdog",
		"scheduled_interval", w.Config.ScheduledGCInterval,
		"pressure_check_interval", w.Config.PressureCheckInterval,
		"pressure_threshold", w.Config.DiskPressureThreshold,
		"retention_days", w.Config.RetentionDays,
		"retention_count", w.Config.RetentionCount)

	ctx, w.cancel = context.WithCancel(ctx)
	go w.monitor(ctx)
}

// Stop gracefully stops the watchdog.
func (w *ImageWatchdog) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

// monitor runs the periodic GC loops.
func (w *ImageWatchdog) monitor(ctx context.Context) {
	pressureTicker := time.NewTicker(w.Config.PressureCheckInterval)
	scheduledTicker := time.NewTicker(w.Config.ScheduledGCInterval)
	defer pressureTicker.Stop()
	defer scheduledTicker.Stop()

	// Run an initial pressure check on startup
	w.checkAndRunGC(ctx, "startup")

	for {
		select {
		case <-pressureTicker.C:
			w.checkAndRunGC(ctx, "pressure_check")
		case <-scheduledTicker.C:
			w.runScheduledGC(ctx)
		case <-ctx.Done():
			w.Log.Info("image watchdog stopped")
			return
		}
	}
}

// checkAndRunGC checks disk pressure and runs GC if threshold is exceeded.
func (w *ImageWatchdog) checkAndRunGC(ctx context.Context, trigger string) {
	stats := sysstats.CollectSystemStats(w.DataPath)
	w.Log.Debug("checking disk pressure",
		"trigger", trigger,
		"storage_percent", stats.StoragePercent,
		"threshold", w.Config.DiskPressureThreshold)

	if stats.StoragePercent >= w.Config.DiskPressureThreshold {
		w.Log.Info("disk pressure threshold exceeded, running GC",
			"storage_percent", stats.StoragePercent,
			"threshold", w.Config.DiskPressureThreshold)
		w.runGCWithLogging(ctx, "disk_pressure")
	}
}

// runScheduledGC runs the weekly scheduled GC.
func (w *ImageWatchdog) runScheduledGC(ctx context.Context) {
	w.Log.Info("running scheduled image GC")
	w.runGCWithLogging(ctx, "scheduled")
}

// runGCWithLogging runs GC and logs results.
func (w *ImageWatchdog) runGCWithLogging(ctx context.Context, trigger string) {
	result, err := w.RunGC(ctx)
	if err != nil {
		w.Log.Error("image GC failed", "trigger", trigger, "error", err)
		return
	}

	if len(result.DeletedImages) > 0 || len(result.FailedImages) > 0 {
		w.Log.Info("image GC complete",
			"trigger", trigger,
			"deleted", len(result.DeletedImages),
			"failed", len(result.FailedImages),
			"retained", result.RetainedImages,
			"total", result.TotalImages)

		for _, img := range result.DeletedImages {
			w.Log.Debug("deleted image", "image", img)
		}
		for img, err := range result.FailedImages {
			w.Log.Warn("failed to delete image", "image", img, "error", err)
		}
	} else {
		w.Log.Debug("image GC complete, no images deleted",
			"trigger", trigger,
			"retained", result.RetainedImages,
			"total", result.TotalImages)
	}
}

// RunGC performs garbage collection of unused images.
func (w *ImageWatchdog) RunGC(ctx context.Context) (*ImageGCResult, error) {
	result := &ImageGCResult{
		DeletedImages: []string{},
		FailedImages:  make(map[string]error),
	}

	gcCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	gcCtx = namespaces.WithNamespace(gcCtx, w.Namespace)

	// List all images
	images, err := w.CC.ListImages(gcCtx)
	if err != nil {
		return result, fmt.Errorf("failed to list images: %w", err)
	}

	result.TotalImages = len(images)

	// Collect images that should be retained
	retainedImages, err := w.collectRetainedImages(gcCtx)
	if err != nil {
		return result, fmt.Errorf("failed to collect retained images: %w", err)
	}

	// Delete images not in retained set
	for _, img := range images {
		imgName := img.Name()

		if retainedImages[imgName] {
			result.RetainedImages++
			continue
		}

		w.Log.Debug("deleting unused image", "image", imgName)
		err := w.CC.ImageService().Delete(gcCtx, imgName)
		if err != nil {
			result.FailedImages[imgName] = err
		} else {
			result.DeletedImages = append(result.DeletedImages, imgName)
		}
	}

	return result, nil
}

// collectRetainedImages returns a set of image names that should be retained.
func (w *ImageWatchdog) collectRetainedImages(ctx context.Context) (map[string]bool, error) {
	retained := make(map[string]bool)

	// Always retain infrastructure images
	retained[imagerefs.Pause] = true
	retained[imagerefs.Etcd] = true
	retained[imagerefs.BuildKit] = true
	retained[imagerefs.Minio] = true
	retained[imagerefs.VictoriaLogs] = true
	retained[imagerefs.VictoriaMetrics] = true
	retained[imagerefs.Miren] = true

	// Collect images from running/pending sandboxes
	inUseImages, err := w.collectInUseImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect in-use images: %w", err)
	}
	for img := range inUseImages {
		retained[img] = true
	}

	// Collect images meeting retention policy
	policyImages, err := w.collectPolicyRetainedImages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect policy-retained images: %w", err)
	}
	for img := range policyImages {
		retained[img] = true
	}

	return retained, nil
}

// collectInUseImages returns images referenced by running/pending sandboxes.
func (w *ImageWatchdog) collectInUseImages(ctx context.Context) (map[string]bool, error) {
	images := make(map[string]bool)

	resp, err := w.EAC.List(ctx, entity.Ref(entity.EntityKind, compute.KindSandbox))
	if err != nil {
		return nil, fmt.Errorf("failed to list sandboxes: %w", err)
	}

	for _, e := range resp.Values() {
		var sb compute.Sandbox
		sb.Decode(e.Entity())

		// Only consider running or pending sandboxes
		if sb.Status != compute.RUNNING && sb.Status != compute.PENDING && sb.Status != "" {
			continue
		}

		// Add all container images
		for _, container := range sb.Spec.Container {
			if container.Image != "" {
				images[container.Image] = true
			}
		}
	}

	w.Log.Debug("collected in-use images", "count", len(images))
	return images, nil
}

// appVersionInfo holds information about an AppVersion for retention decisions.
type appVersionInfo struct {
	ID        entity.Id
	App       entity.Id
	ImageUrl  string
	CreatedAt time.Time
}

// collectPolicyRetainedImages returns images that meet the retention policy:
// - Images from AppVersions < RetentionDays old
// - Images within the last RetentionCount AppVersions per app
func (w *ImageWatchdog) collectPolicyRetainedImages(ctx context.Context) (map[string]bool, error) {
	images := make(map[string]bool)

	// List all AppVersions
	resp, err := w.EAC.List(ctx, entity.Ref(entity.EntityKind, core_v1alpha.KindAppVersion))
	if err != nil {
		return nil, fmt.Errorf("failed to list app versions: %w", err)
	}

	now := time.Now()
	retentionCutoff := now.Add(-time.Duration(w.Config.RetentionDays) * 24 * time.Hour)

	// Group versions by app
	versionsByApp := make(map[entity.Id][]appVersionInfo)

	for _, e := range resp.Values() {
		var av core_v1alpha.AppVersion
		av.Decode(e.Entity())

		if av.ImageUrl == "" {
			continue
		}

		info := appVersionInfo{
			ID:        av.ID,
			App:       av.App,
			ImageUrl:  av.ImageUrl,
			CreatedAt: e.Entity().GetCreatedAt(),
		}

		// Always retain if within retention days
		if info.CreatedAt.After(retentionCutoff) {
			images[av.ImageUrl] = true
		}

		// Group by app for count-based retention
		if av.App != "" {
			versionsByApp[av.App] = append(versionsByApp[av.App], info)
		}
	}

	// Apply count-based retention: keep last N versions per app
	for _, versions := range versionsByApp {
		// Sort by creation time, newest first
		sort.Slice(versions, func(i, j int) bool {
			return versions[i].CreatedAt.After(versions[j].CreatedAt)
		})

		// Keep the most recent RetentionCount versions
		for i, v := range versions {
			if i >= w.Config.RetentionCount {
				break
			}
			images[v.ImageUrl] = true
		}
	}

	w.Log.Debug("collected policy-retained images",
		"count", len(images),
		"apps", len(versionsByApp))

	return images, nil
}
