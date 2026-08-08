package app

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/fake"
)

func TestConfigMapMetricsCacheCreatesAndLoadsSnapshot(t *testing.T) {
	cache := &ConfigMapMetricsCache{
		configMaps: fake.NewClientset(),
		namespace:  "backup",
		name:       "borg-exporter-metrics-cache",
	}
	snapshot := MetricsSnapshot{
		Metrics: "borg_backups_total 42\n",
		SavedAt: time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC),
	}
	if err := cache.Save(context.Background(), snapshot); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	actual, found, err := cache.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !found {
		t.Fatal("Load() found = false, want true")
	}
	if actual != snapshot {
		t.Fatalf("Load() = %#v, want %#v", actual, snapshot)
	}
}

func TestWriteSnapshotAddsFreshnessMetrics(t *testing.T) {
	recorder := httptest.NewRecorder()
	snapshot := MetricsSnapshot{
		Metrics: "borg_backups_total 42\n",
		SavedAt: time.Now().Add(-time.Minute),
	}
	writeSnapshot(recorder, snapshot)

	body := recorder.Body.String()
	for _, expected := range []string{
		"borg_backups_total 42",
		"borg_exporter_snapshot_last_success_timestamp_seconds",
		"borg_exporter_snapshot_age_seconds",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %q: %s", expected, body)
		}
	}
}

func TestConfigMapMetricsCacheMissingSnapshot(t *testing.T) {
	cache := &ConfigMapMetricsCache{
		configMaps: fake.NewClientset(),
		namespace:  "backup",
		name:       "borg-exporter-metrics-cache",
	}
	_, found, err := cache.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if found {
		t.Fatal("Load() found = true, want false")
	}
}
