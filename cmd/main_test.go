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
