// tests/admin_login_test.go
//go:build integration

package test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAdminLogin verifies that the AdminLogin function works correctly
func TestAdminLogin(t *testing.T) {
	// Create test harness
	sim := NewSimulationTest(t, testMeetName)
	defer sim.Close()

	// Test admin login - this should succeed
	sim.AdminLogin()

	// If we got this far without errors, the test passes
	require.NotNil(t, sim.cookie, "Cookie should be set after login")
}
