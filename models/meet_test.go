// file: models/meet_test.go
package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ✅ Test: Create a User and verify struct fields
func TestUserInitialization(t *testing.T) {
	user := User{
		Username: "testuser",
		Password: "securepassword",
	}

	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "securepassword", user.Password)
}

// ✅ Test: Create a Meet and verify struct fields
func TestMeetInitialization(t *testing.T) {
	users := []User{
		{Username: "referee1", Password: "pass1"},
		{Username: "referee2", Password: "pass2"},
	}

	meet := Meet{
		Name:  "State Powerlifting Championship",
		Date:  "2025-03-15",
		Users: users,
	}

	assert.Equal(t, "State Powerlifting Championship", meet.Name)
	assert.Equal(t, "2025-03-15", meet.Date)
	assert.Len(t, meet.Users, 2)
}

// ✅ Test: Create MeetCreds and verify multiple meets
func TestMeetCredsInitialization(t *testing.T) {
	meet1 := Meet{Name: "Nationals", Date: "2025-06-20"}
	meet2 := Meet{Name: "Regionals", Date: "2025-04-10"}

	meetCreds := MeetCreds{
		Meets: []Meet{meet1, meet2},
	}

	assert.Len(t, meetCreds.Meets, 2)
	assert.Equal(t, "Nationals", meetCreds.Meets[0].Name)
	assert.Equal(t, "2025-04-10", meetCreds.Meets[1].Date)
}

// ✅ Test: User JSON Serialization & Deserialization
func TestUserJSONSerialization(t *testing.T) {
	user := User{Username: "testuser", Password: "securepass"}

	// ✅ Serialize User to JSON
	jsonData, err := json.Marshal(user)
	assert.NoError(t, err)

	// ✅ Deserialize JSON back into User struct
	var decodedUser User
	err = json.Unmarshal(jsonData, &decodedUser)
	assert.NoError(t, err)

	// ✅ Verify data integrity
	assert.Equal(t, user.Username, decodedUser.Username)
	assert.Equal(t, user.Password, decodedUser.Password)
}

// ✅ Test: Meet JSON Serialization & Deserialization
func TestMeetJSONSerialization(t *testing.T) {
	users := []User{{Username: "ref1", Password: "pass1"}}
	meet := Meet{Name: "Deadlift Open", Date: "2025-05-01", Users: users}

	// ✅ Serialize Meet to JSON
	jsonData, err := json.Marshal(meet)
	assert.NoError(t, err)

	// ✅ Deserialize JSON back into Meet struct
	var decodedMeet Meet
	err = json.Unmarshal(jsonData, &decodedMeet)
	assert.NoError(t, err)

	// ✅ Verify data integrity
	assert.Equal(t, meet.Name, decodedMeet.Name)
	assert.Equal(t, meet.Date, decodedMeet.Date)
	assert.Len(t, decodedMeet.Users, 1)
	assert.Equal(t, meet.Users[0].Username, decodedMeet.Users[0].Username)
}

// ✅ Test: MeetCreds JSON Serialization & Deserialization
func TestMeetCredsJSONSerialization(t *testing.T) {
	meetCreds := MeetCreds{
		Meets: []Meet{
			{Name: "Nationals", Date: "2025-06-20"},
			{Name: "Regionals", Date: "2025-04-10"},
		},
	}

	// ✅ Serialize MeetCreds to JSON
	jsonData, err := json.Marshal(meetCreds)
	assert.NoError(t, err)

	// ✅ Deserialize JSON back into MeetCreds struct
	var decodedMeetCreds MeetCreds
	err = json.Unmarshal(jsonData, &decodedMeetCreds)
	assert.NoError(t, err)

	// ✅ Verify data integrity
	assert.Len(t, decodedMeetCreds.Meets, 2)
	assert.Equal(t, "Nationals", decodedMeetCreds.Meets[0].Name)
	assert.Equal(t, "2025-04-10", decodedMeetCreds.Meets[1].Date)
}
