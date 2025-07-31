package testutils

import (
	"testing"
)

// TestDummy is a simple test to prevent "no test files" error
func TestDummy(t *testing.T) {
	// This test ensures the testutils package has at least one test
	t.Log("testutils package test file present")
}