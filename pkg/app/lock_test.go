package app

import "testing"

func TestNewRepoLockFromEnvDefaults(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "")
	t.Setenv("BORG_LOCK_NAME", "")
	t.Setenv("POD_UID", "")
	t.Setenv("HOSTNAME", "")

	lock := NewRepoLockFromEnv()
	if lock.namespace != defaultLockNamespace {
		t.Fatalf("namespace = %q, want %q", lock.namespace, defaultLockNamespace)
	}
	if lock.name != defaultLockName {
		t.Fatalf("name = %q, want %q", lock.name, defaultLockName)
	}
	if lock.identity != "borg-exporter" {
		t.Fatalf("identity = %q, want borg-exporter", lock.identity)
	}
}

func TestNewRepoLockFromEnvUsesPodIdentity(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "production")
	t.Setenv("BORG_LOCK_NAME", "repository-lock")
	t.Setenv("POD_UID", "pod-uid")
	t.Setenv("HOSTNAME", "ignored-hostname")

	lock := NewRepoLockFromEnv()
	if lock.namespace != "production" || lock.name != "repository-lock" || lock.identity != "pod-uid" {
		t.Fatalf("lock = %+v, want configured values", lock)
	}
}
