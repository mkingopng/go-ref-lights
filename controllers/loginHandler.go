// Package controllers: controllers/loginHandler.go
package controllers

import (
	"golang.org/x/crypto/bcrypt"
)

// compare hashed password
func checkPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
