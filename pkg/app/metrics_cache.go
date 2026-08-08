package app

import (
	"context"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	metricsCacheNameEnv = "BORG_METRICS_CACHE_CONFIGMAP"
	metricsDataKey      = "metrics"
	savedAtDataKey      = "saved_at"
)

// MetricsCache retains the last complete repository snapshot. It must never
// contain credentials, repository state, or Borg cache data.
type MetricsCache interface {
	Load(context.Context) (MetricsSnapshot, bool, error)
	Save(context.Context, MetricsSnapshot) error
}

type MetricsSnapshot struct {
	Metrics string
	SavedAt time.Time
}

type ConfigMapMetricsCache struct {
	configMaps kubernetes.Interface
	namespace  string
	name       string
}

// NewConfigMapMetricsCacheFromEnv enables persisted snapshots only when the
// ConfigMap name is configured. This preserves the existing behavior for
// deployments that have not granted ConfigMap RBAC.
func NewConfigMapMetricsCacheFromEnv() (MetricsCache, error) {
	name := os.Getenv(metricsCacheNameEnv)
	if name == "" {
		return nil, nil
	}
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = defaultLockNamespace
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("load in-cluster Kubernetes configuration for metrics cache: %w", err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes client for metrics cache: %w", err)
	}
	return &ConfigMapMetricsCache{configMaps: client, namespace: namespace, name: name}, nil
}

func (c *ConfigMapMetricsCache) Load(ctx context.Context) (MetricsSnapshot, bool, error) {
	configMap, err := c.configMaps.CoreV1().ConfigMaps(c.namespace).Get(ctx, c.name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return MetricsSnapshot{}, false, nil
	}
	if err != nil {
		return MetricsSnapshot{}, false, fmt.Errorf("get metrics cache ConfigMap: %w", err)
	}
	savedAt, err := time.Parse(time.RFC3339Nano, configMap.Data[savedAtDataKey])
	if err != nil {
		return MetricsSnapshot{}, false, fmt.Errorf("parse metrics cache timestamp: %w", err)
	}
	metrics := configMap.Data[metricsDataKey]
	if metrics == "" {
		return MetricsSnapshot{}, false, nil
	}
	return MetricsSnapshot{Metrics: metrics, SavedAt: savedAt}, true, nil
}

func (c *ConfigMapMetricsCache) Save(ctx context.Context, snapshot MetricsSnapshot) error {
	configMaps := c.configMaps.CoreV1().ConfigMaps(c.namespace)
	data := map[string]string{
		metricsDataKey: metricsCacheData(snapshot),
		savedAtDataKey: snapshot.SavedAt.UTC().Format(time.RFC3339Nano),
	}
	configMap, err := configMaps.Get(ctx, c.name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = configMaps.Create(ctx, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: c.name},
			Data:       data,
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create metrics cache ConfigMap: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get metrics cache ConfigMap for update: %w", err)
	}
	configMap.Data = data
	if _, err := configMaps.Update(ctx, configMap, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update metrics cache ConfigMap: %w", err)
	}
	return nil
}

func metricsCacheData(snapshot MetricsSnapshot) string {
	return snapshot.Metrics
}
