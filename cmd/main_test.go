package main

import (
	"strings"
	"testing"
)

func TestLockRunRequiresCommand(t *testing.T) {
	err := lockRun(nil)
	if err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("lockRun(nil) error = %v, want usage error", err)
	}
}

func TestNewAppInitializesRepoLock(t *testing.T) {
	app := newApp("/etc/borgmatic.d", "9996", nil)
	if app.RepoLock == nil {
		t.Fatal("RepoLock must be initialized")
	}
}
