// Package models defines data structures used across the application.
// File: models/meet.go
package models

// ----------------------- user model -----------------------

// User represents an individual user with a username and password in the system
type User struct {
	Username string `json:"username"` // User's unique identifier
	Password string `json:"password"` // User's password (hashed in production)
	IsAdmin  bool   `json:"isadmin"`  // Admin privilege flag
}

// ------------------------ meet model -----------------------

// Meet represents a powerlifting meet with associated users.
type Meet struct {
	Name  string `json:"name"`  // Meet name
	Date  string `json:"date"`  // Meet date (should use time.Time in production)
	Users []User `json:"users"` // List of registered users
}

// ---------------------- meet credentials model ----------------------

// MeetCreds holds a collection of powerlifting meets.
type MeetCreds struct {
	Meets []Meet `json:"meets"` // List of meets
}
