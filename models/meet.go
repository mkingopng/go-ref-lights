// Package models defines data structures used across the application.
// File: models/meet.go
package models

// ----------------------- user model -----------------------

// Admin represents the meet admin user.
type Admin struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"isadmin"`
}

// ------------------------ meet model -----------------------

// Meet represents a powerlifting meet with associated users.
type Meet struct {
	Name  string `json:"name"`  // Meet name
	Date  string `json:"date"`  // Meet date (should use time.Time in production)
	Admin Admin  `json:"admin"` // Meet admin user
	Logo  string `json:"logo"`  // Meet logo URL
}

// ---------------------- meet credentials model ----------------------

// MeetCreds holds a collection of powerlifting meets.
type MeetCreds struct {
	Meets []Meet `json:"meets"` // List of meets
}
