package app

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	defaultLockName      = "borg-repository"
	defaultLockNamespace = "backup"
)

// RepoLock serializes every Borg repository access, including read-only info
// calls. A Lease expires if its holder is killed, so a failed pod cannot leave
// the repository locked forever.
type RepoLock struct {
	name      string
	namespace string
	identity  string
}

func NewRepoLockFromEnv() *RepoLock {
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		namespace = defaultLockNamespace
	}

	name := os.Getenv("BORG_LOCK_NAME")
	if name == "" {
		name = defaultLockName
	}

	identity := os.Getenv("POD_UID")
	if identity == "" {
		identity = os.Getenv("HOSTNAME")
	}
	if identity == "" {
		identity = "borg-exporter"
	}

	return &RepoLock{name: name, namespace: namespace, identity: identity}
}

// With runs operation only while this process owns the shared Kubernetes Lease.
// The operation context is canceled immediately if Lease renewal fails.
func (l *RepoLock) With(ctx context.Context, operation func(context.Context) error) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("load in-cluster Kubernetes configuration: %w", err)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("create Kubernetes client: %w", err)
	}

	lock, err := resourcelock.New(
		resourcelock.LeasesResourceLock,
		l.namespace,
		l.name,
		client.CoreV1(),
		client.CoordinationV1(),
		resourcelock.ResourceLockConfig{Identity: l.identity},
	)
	if err != nil {
		return fmt.Errorf("create repository Lease lock: %w", err)
	}

	leaderCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	result := make(chan error, 1)
	electionDone := make(chan struct{})
	var completed atomic.Bool

	go func() {
		defer close(electionDone)
		leaderelection.RunOrDie(leaderCtx, leaderelection.LeaderElectionConfig{
			Lock:            lock,
			LeaseDuration:   60 * time.Second,
			RenewDeadline:   40 * time.Second,
			RetryPeriod:     10 * time.Second,
			ReleaseOnCancel: true,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(leaderCtx context.Context) {
					err := operation(leaderCtx)
					completed.Store(true)
					result <- err
					cancel()
				},
				OnStoppedLeading: func() {
					if !completed.Load() {
						result <- fmt.Errorf("lost repository Lease before operation completed")
					}
				},
			},
		})
	}()

	select {
	case err := <-result:
		<-electionDone
		return err
	case <-ctx.Done():
		<-electionDone
		return ctx.Err()
	}
}
