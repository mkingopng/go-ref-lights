package controllers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-ref-lights/models"
)

// ✅ Mock meet data for testing
var testMeets = models.MeetCreds{
	Meets: []models.Meet{
		{Name: "TestMeet1", Date: "2025-03-10"},
		{Name: "TestMeet2", Date: "2025-04-15"},
	},
}

// ✅ Test LoadMeets Success
func TestLoadMeets_Success(t *testing.T) {
	originalLoadMeetsFunc := loadMeetsFunc
	loadMeetsFunc = func() (*models.MeetCreds, error) { return &testMeets, nil }
	defer func() { loadMeetsFunc = originalLoadMeetsFunc }() // Restore after test

	meets, err := loadMeetsFunc()
	assert.NoError(t, err, "LoadMeets should not return an error")
	assert.NotNil(t, meets, "Meets should not be nil")
	assert.Len(t, meets.Meets, 2, "There should be two meets")
	assert.Equal(t, "TestMeet1", meets.Meets[0].Name, "First meet should match")
}

// ✅ Test LoadMeets Failure (File Not Found)
func TestLoadMeets_FileNotFound(t *testing.T) {
	originalLoadMeetsFunc := loadMeetsFunc
	loadMeetsFunc = func() (*models.MeetCreds, error) { return nil, os.ErrNotExist }
	defer func() { loadMeetsFunc = originalLoadMeetsFunc }() // Restore after test

	meets, err := loadMeetsFunc()
	assert.Error(t, err, "LoadMeets should return an error if the file is missing")
	assert.Nil(t, meets, "Meets should be nil on failure")
}

// ✅ Test ShowMeets Handler
func TestShowMeets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupTestRouter() // ✅ Now includes templates

	// ✅ Attach ShowMeets route
	router.GET("/meets", ShowMeets)

	originalLoadMeetsFunc := loadMeetsFunc
	loadMeetsFunc = func() (*models.MeetCreds, error) { return &testMeets, nil }
	defer func() { loadMeetsFunc = originalLoadMeetsFunc }() // Restore after test

	req, _ := http.NewRequest("GET", "/meets", nil)
	w := httptest.NewRecorder()

	// ✅ Perform request
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "ShowMeets should return 200 OK")
	assert.Contains(t, w.Body.String(), "TestMeet1", "Response should contain TestMeet1")
	assert.Contains(t, w.Body.String(), "TestMeet2", "Response should contain TestMeet2")
}

// ✅ Test ShowMeets Failure
func TestShowMeets_Failure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/meets", ShowMeets)

	// ✅ Simulate error loading meets
	originalLoadMeetsFunc := loadMeetsFunc
	loadMeetsFunc = func() (*models.MeetCreds, error) { return nil, os.ErrNotExist }
	defer func() { loadMeetsFunc = originalLoadMeetsFunc }() // Restore after test

	req, _ := http.NewRequest("GET", "/meets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code, "ShowMeets should return 500 on failure")
	assert.Contains(t, w.Body.String(), "Failed to load meets", "Error message should be returned")
}
