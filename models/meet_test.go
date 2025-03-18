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
func TestAdminInitialization(t *testing.T) {
	admin := Admin{
		Username: "testadmin",
		Password: "securepassword",
		IsAdmin:  true,
	}
	assert.Equal(t, "testadmin", admin.Username)
	assert.Equal(t, "securepassword", admin.Password)
	assert.True(t, admin.IsAdmin)
}

// Test: Create a Meet and verify struct fields
func TestMeetInitialization(t *testing.T) {
	websocket.InitTest()

	adminUser := Admin{
		Username: "adminUser",
		Password: "adminPass",
		IsAdmin:  true,
	}
	meet := Meet{
		Name:  "State Powerlifting Championship",
		Date:  "2025-03-15",
		Admin: adminUser,
		Logo:  "championship_logo.png",
	}

	assert.Equal(t, "State Powerlifting Championship", meet.Name)
	assert.Equal(t, "2025-03-15", meet.Date)
	assert.Equal(t, "adminUser", meet.Admin.Username)
	assert.Equal(t, "adminPass", meet.Admin.Password)
	assert.True(t, meet.Admin.IsAdmin)
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

// Test: Meet JSON Serialization & Deserialization
func TestMeetJSONSerialization(t *testing.T) {
	websocket.InitTest()

	meet := Meet{
		Name: "Deadlift Open",
		Date: "2025-05-01",
		Admin: Admin{
			Username: "adminUser",
			Password: "adminPass",
			IsAdmin:  true,
		},
		Logo: "deadlift_open_logo.png",
	}

	// Serialize Meet to JSON.
	jsonData, err := json.Marshal(meet)
	assert.NoError(t, err)

	// Deserialize JSON back into Meet struct.
	var decodedMeet Meet
	err = json.Unmarshal(jsonData, &decodedMeet)
	assert.NoError(t, err)

	// Verify data integrity.
	assert.Equal(t, meet.Name, decodedMeet.Name)
	assert.Equal(t, meet.Date, decodedMeet.Date)
	assert.Equal(t, meet.Admin.Username, decodedMeet.Admin.Username)
	assert.Equal(t, meet.Admin.Password, decodedMeet.Admin.Password)
	assert.Equal(t, meet.Admin.IsAdmin, decodedMeet.Admin.IsAdmin)
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
