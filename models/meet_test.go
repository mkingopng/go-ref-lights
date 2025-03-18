// file: models/meet_test.go

//go:build unit
// +build unit

package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"go-ref-lights/websocket"
)

// Test: Create a User and verify struct fields
func TestUserInitialization(t *testing.T) {
	user := User{
		Username: "testuser",
		Password: "securepassword",
	}

	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "securepassword", user.Password)
}

// Test: Create a Meet and verify struct fields
func TestMeetInitialization(t *testing.T) {
	websocket.InitTest()

	// --- REMOVED: old slice of users ---
	// users := []User{
	//     {Username: "referee1", Password: "pass1"},
	//     {Username: "referee2", Password: "pass2"},
	// }

	// --- ADDED: define Admin and single User to match new struct ---
	adminUser := User{
		Username: "adminUser",
		Password: "adminPass",
		IsAdmin:  true,
	}
	normalUser := User{
		Username: "referee1",
		Password: "pass1",
		IsAdmin:  false,
	}

	// --- CHANGED: now we specify Admin, User, Logo instead of Users []User ---
	meet := Meet{
		Name:  "State Powerlifting Championship",
		Date:  "2025-03-15",
		Admin: adminUser,
		User:  normalUser,
		Logo:  "championship_logo.png",
	}

	assert.Equal(t, "State Powerlifting Championship", meet.Name)
	assert.Equal(t, "2025-03-15", meet.Date)
	assert.Equal(t, "adminUser", meet.Admin.Username)
	assert.Equal(t, "adminPass", meet.Admin.Password)
	assert.True(t, meet.Admin.IsAdmin)
	assert.Equal(t, "referee1", meet.User.Username)
	assert.Equal(t, "pass1", meet.User.Password)
	assert.False(t, meet.User.IsAdmin)
	assert.Equal(t, "championship_logo.png", meet.Logo)
}

// Test: Create MeetCreds and verify multiple meets
func TestMeetCredsInitialization(t *testing.T) {
	websocket.InitTest()
	meet1 := Meet{Name: "Nationals", Date: "2025-06-20"}
	meet2 := Meet{Name: "Regionals", Date: "2025-04-10"}

	meetCreds := MeetCreds{
		Meets: []Meet{meet1, meet2},
	}

	assert.Len(t, meetCreds.Meets, 2)
	assert.Equal(t, "Nationals", meetCreds.Meets[0].Name)
	assert.Equal(t, "2025-04-10", meetCreds.Meets[1].Date)
}

// Test: User JSON Serialization & Deserialization
func TestUserJSONSerialization(t *testing.T) {
	websocket.InitTest()
	user := User{Username: "testuser", Password: "securepass"}

	// Serialize User to JSON
	jsonData, err := json.Marshal(user)
	assert.NoError(t, err)

	// Deserialize JSON back into User struct
	var decodedUser User
	err = json.Unmarshal(jsonData, &decodedUser)
	assert.NoError(t, err)

	// Verify data integrity
	assert.Equal(t, user.Username, decodedUser.Username)
	assert.Equal(t, user.Password, decodedUser.Password)
}

// Test: Meet JSON Serialization & Deserialization
func TestMeetJSONSerialization(t *testing.T) {
	websocket.InitTest()

	// --- REMOVED: old slice of users ---
	// users := []User{{Username: "ref1", Password: "pass1"}}

	// --- ADDED: define Admin and single User to match new struct ---
	meet := Meet{
		Name: "Deadlift Open",
		Date: "2025-05-01",
		Admin: User{
			Username: "adminUser",
			Password: "adminPass",
			IsAdmin:  true,
		},
		User: User{
			Username: "ref1",
			Password: "pass1",
			IsAdmin:  false,
		},
		Logo: "deadlift_open_logo.png",
	}

	// Serialize Meet to JSON
	jsonData, err := json.Marshal(meet)
	assert.NoError(t, err)

	// Deserialize JSON back into Meet struct
	var decodedMeet Meet
	err = json.Unmarshal(jsonData, &decodedMeet)
	assert.NoError(t, err)

	// Verify data integrity
	assert.Equal(t, meet.Name, decodedMeet.Name)
	assert.Equal(t, meet.Date, decodedMeet.Date)
	assert.Equal(t, meet.Admin.Username, decodedMeet.Admin.Username)
	assert.Equal(t, meet.Admin.Password, decodedMeet.Admin.Password)
	assert.Equal(t, meet.Admin.IsAdmin, decodedMeet.Admin.IsAdmin)
	assert.Equal(t, meet.User.Username, decodedMeet.User.Username)
	assert.Equal(t, meet.User.Password, decodedMeet.User.Password)
	assert.Equal(t, meet.User.IsAdmin, decodedMeet.User.IsAdmin)
	assert.Equal(t, meet.Logo, decodedMeet.Logo)
}

// Test: MeetCreds JSON Serialization & Deserialization
func TestMeetCredsJSONSerialization(t *testing.T) {
	websocket.InitTest()
	meetCreds := MeetCreds{
		Meets: []Meet{
			{Name: "Nationals", Date: "2025-06-20"},
			{Name: "Regionals", Date: "2025-04-10"},
		},
	}

	// Serialize MeetCreds to JSON
	jsonData, err := json.Marshal(meetCreds)
	assert.NoError(t, err)

	// Deserialize JSON back into MeetCreds struct
	var decodedMeetCreds MeetCreds
	err = json.Unmarshal(jsonData, &decodedMeetCreds)
	assert.NoError(t, err)

	// Verify data integrity
	assert.Len(t, decodedMeetCreds.Meets, 2)
	assert.Equal(t, "Nationals", decodedMeetCreds.Meets[0].Name)
	assert.Equal(t, "2025-04-10", decodedMeetCreds.Meets[1].Date)
}
