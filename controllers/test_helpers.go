//go:build unit
// +build unit

// file: controllers/test_helpers.go
package controllers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"go-ref-lights/services"
	"golang.org/x/crypto/bcrypt"
)

// -------------------- mock service for testing --------------------

// MockOccupancyService is a mock implementation of the OccupancyServiceInterface.
type MockOccupancyService struct {
	mock.Mock
}

// ---------------------- mock method implementations ----------------------

// UnsetPosition removes the position assignment for a given referee.
func (m *MockOccupancyService) UnsetPosition(meetName, position, user string) error {
	args := m.Called(meetName, position, user)
	return args.Error(0)
}

// GetOccupancy retrieves the current occupancy status for a given meet.
func (m *MockOccupancyService) GetOccupancy(meetName string) services.Occupancy {
	args := m.Called(meetName)
	return args.Get(0).(services.Occupancy)
}

// ResetOccupancyForMeet clears all referee positions for a specific meet.
func (m *MockOccupancyService) ResetOccupancyForMeet(meetName string) {
	m.Called(meetName)
}

// SetPosition assigns a referee to a specific position in a meet.
func (m *MockOccupancyService) SetPosition(meetName, position, user string) error {
	args := m.Called(meetName, position, user)
	return args.Error(0)
}

type MockPositionController struct {
	mock.Mock
}

func (mp *MockPositionController) BroadcastOccupancy(meetName string) {
	mp.Called(meetName)
}

// -------------------------

// setupTestRouter creates a new Gin engine with session middleware and fake HTML templates
func setupTestRouter(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// set up sessions with cookie store
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("mySession", store))

	// create minimal templates to avoid panics during testing
	tmpDir := t.TempDir()
	if err := createDummyTemplates(tmpDir); err != nil {
		t.Fatalf("Failed to create dummy templates: %v", err)
	}

	// use filepath.Join for cross-platform compatibility
	router.LoadHTMLGlob(filepath.Join(tmpDir, "*.html"))
	return router
}

// createDummyTemplates writes a set of minimal HTML templates to the provided directory
func createDummyTemplates(dir string) error {
	templates := map[string]string{
		"choose_meet.html": `<html><body>{{.}}</body></html>`,
		"login.html":       `<html><body>{{.}}</body></html>`,
		"index.html":       `<html><body>{{.}}</body></html>`,
		"left.html":        `<html><body>Left ref view for {{.meetName}}</body></html>`,
		"center.html":      `<html><body>Center ref view for {{.meetName}}</body></html>`,
		"right.html":       `<html><body>Right ref view for {{.meetName}}</body></html>`,
		"admin.html":       `<html><body>Admin panel for {{.meetName}}</body></html>`,
	}

	for name, content := range templates {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			return err
		}
	}
	return nil
}

// SetSession sets the given key/value pairs in the session using a helper route
// and returns the session cookie that can be attached to subsequent test requests
func SetSession(router *gin.Engine, route string, data map[string]interface{}) *http.Cookie {
	// create a helper route for setting session values.
	router.GET(route, func(c *gin.Context) {
		session := sessions.Default(c)
		for key, value := range data {
			session.Set(key, value)
		}
		if err := session.Save(); err != nil {
			c.String(http.StatusInternalServerError, "session save failed")
			return
		}
		c.String(http.StatusOK, "session set")
	})

	// call the helper route
	parsedURL, err := url.Parse(route)
	if err != nil || parsedURL.Host != "" {
		return nil
	}

	// create a request to the helper route
	req, _ := http.NewRequest("GET", parsedURL.Path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// extract and return the session cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "mySession" {
			return cookie
		}
		return cookie
	}
	return nil
}

// hashPassword hashes the given password using bcrypt
func hashPassword(password string) string {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic("failed to hash password: " + err.Error())
	}
	return string(hashed)
}

// createPostRequest creates a new POST request with the given form data
func createPostRequest(path string, formData map[string]string) *http.Request {
	form := url.Values{}
	for k, v := range formData {
		form.Add(k, v)
	}
	req := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

// performRequest executes the given request on the provided router and returns the response
func performRequest(router http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
