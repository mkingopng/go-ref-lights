// file: controllers/loginHandler_test.go

//go:build unit
// +build unit

package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go-ref-lights/websocket"
	"golang.org/x/crypto/bcrypt"
)

// TestCheckPasswordHash tests the checkPasswordHash function
func TestCheckPasswordHash(t *testing.T) {
	websocket.InitTest()
	// Define a plaintext password
	password := "securepassword123"

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err, "Password hashing should not return an error")

	// Test: Correct password should return true
	assert.True(t, checkPasswordHash(password, string(hashedPassword)), "Correct password should match the hash")

	// Test: Incorrect password should return false
	assert.False(t, checkPasswordHash("wrongpassword", string(hashedPassword)), "Incorrect password should not match the hash")

	// Test: Empty password should return false
	assert.False(t, checkPasswordHash("", string(hashedPassword)), "Empty password should not match the hash")

	// Test: Empty hash should return false
	assert.False(t, checkPasswordHash(password, ""), "Valid password should not match an empty hash")
}
