// Package models - models/meet.go
package models

// User represents an individual user with a username and password
type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Meet represents a powerlifting meet with associated users
type Meet struct {
	Name  string `json:"name"`
	Date  string `json:"date"`
	Users []User `json:"users"`
}

// MeetCreds stores multiple meets
type MeetCreds struct {
	Meets []Meet `json:"meets"`
}
