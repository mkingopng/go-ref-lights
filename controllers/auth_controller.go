// Package controllers provides authentication and session management for users.
// File: controllers/auth_controller.go
package controllers

import (
	"github.com/aws/aws-xray-sdk-go/xray"
	"go-ref-lights/services"
	"net/http"
	"sync"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/models"
	"golang.org/x/crypto/bcrypt"
)

// ---------- global variables ----------

// ActiveUsers tracks currently logged-in users.
var ActiveUsers = make(map[string]bool)

// ActiveUsersMu controls concurrency for ActiveUsers.
var ActiveUsersMu sync.RWMutex

// ----------------------- authentication utilities -----------------------

// lockActiveUsers locks the ActiveUsers map for testing.
//
//nolint:unused
func lockActiveUsers() {
	ActiveUsersMu.Lock()
}

// unlockActiveUsers unlocks the ActiveUsers map for testing.
//
//nolint:unused
func unlockActiveUsers() {
	ActiveUsersMu.Unlock()
}

// setUserActive sets a user as active for testing.
//
//nolint:unused
func setUserActive(username string) {
	ActiveUsers[username] = true
}

// clearUserActive clears a user from the active users map for testing.
//
//nolint:unused
func clearUserActive(username string) {
	delete(ActiveUsers, username)
}

// ComparePasswords checks if the given password matches the hashed password
func ComparePasswords(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

// SetMeetHandler sets the selected meet in the session and redirects to the meet page.
func SetMeetHandler(c *gin.Context) {
	meetName := c.PostForm("meetName")
	if meetName == "" {
		c.HTML(http.StatusBadRequest, "choose_meet.html", gin.H{"Error": "Please select a meet."})
		return
	}
	session := sessions.Default(c)
	session.Set("meetName", meetName)
	if err := session.Save(); err != nil {
		logger.Error.Printf("Failed to save meet session: %v", err)
		c.HTML(http.StatusInternalServerError, "choose_meet.html", gin.H{"Error": "Internal error, please try again."})
		return
	}
	logger.Info.Printf("Meet %s selected, redirecting to meet page.", meetName)
	c.Redirect(http.StatusFound, "/login")
}

// ----------------------- meet selection ----------------------------------

// MeetHandler retrieves the meet details from session and renders the home page with the appropriate logo.
func MeetHandler(c *gin.Context) {
	session := sessions.Default(c)
	storedMeet := session.Get("meetName")
	if storedMeet == nil {
		c.HTML(http.StatusBadRequest, "choose_meet.html", gin.H{"Error": "No meet selected."})
		return
	}
	meetName := storedMeet.(string)

	// load meet credentials using the injectable function.
	creds := services.GetGlobalMeetCredentials()
	if creds == nil {
		logger.Error.Printf("Failed to load meets: %v", creds)
		c.HTML(http.StatusInternalServerError, "choose_meet.html", gin.H{"Error": "Internal error loading meets."})
		return
	}

	// find the meet with the matching name.
	var currentMeet *models.Meet
	for _, meet := range creds.Meets {
		if meet.Name == meetName {
			meetCopy := meet
			currentMeet = &meetCopy
			break
		}
	}
	if currentMeet == nil {
		c.HTML(http.StatusNotFound, "choose_meet.html", gin.H{"Error": "Meet not found."})
		return
	}

	// prepare data for the template.
	data := gin.H{
		"meetName": currentMeet.Name,
		"logo":     currentMeet.Logo,
	}

	// render the template with the correct logo.
	c.HTML(http.StatusOK, "index.html", data)
}

// ----------------------- admin actions -----------------------------------

// ForceLogoutHandler forcibly logs out a user (admin action).
func ForceLogoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	isAdmin := session.Get("isAdmin")

	if isAdmin == nil || isAdmin != true {
		logger.Warn.Println("Unauthorized attempt to force logout a user.")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin privileges required"})
		return
	}

	username := c.PostForm("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing username parameter"})
		return
	}

	// acquire the write lock for read-check + deletion
	ActiveUsersMu.Lock()
	defer ActiveUsersMu.Unlock()

	if _, exists := ActiveUsers[username]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not logged in"})
		return
	}

	delete(ActiveUsers, username)
	logger.Info.Printf("Admin forcibly logged out user: %s", username)

	c.JSON(http.StatusOK, gin.H{"message": "User logged out successfully"})
}

// --------------------- active user tracking ------------------------------

// ActiveUsersHandler returns a list of currently active users (admin action).
func ActiveUsersHandler(c *gin.Context) {
	session := sessions.Default(c)
	isAdmin := session.Get("isAdmin")

	if isAdmin == nil || isAdmin != true {
		logger.Warn.Println("Unauthorized attempt to view active users.")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin privileges required"})
		return
	}

	var userList []string

	// acquire read lock for iteration
	ActiveUsersMu.RLock()
	for user := range ActiveUsers {
		userList = append(userList, user)
	}
	ActiveUsersMu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"users": userList})
}

var occupancyService services.OccupancyServiceInterface

// ------------------ authentication utilities ------------------

// checkPasswordHash verifies if the provided plain-text password matches the stored hashed password.
func checkPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// PerformLogin captures meetName & position from query params for the login page
func PerformLogin(c *gin.Context) {
	session := sessions.Default(c)

	meetNameParam := c.Query("meetName")
	posParam := c.Query("position")

	// check if there's already a different meetName in the session
	if existingMeet, ok := session.Get("meetName").(string); ok && existingMeet != "" && meetNameParam != "" {
		if meetNameParam != existingMeet {
			// we decide to ABORT if there's a conflict
			logger.Warn.Printf("[PerformLogin] Session meetName=%s, but user tried to pass meetName=%s. Conflict => aborting.",
				existingMeet, meetNameParam)
			c.String(http.StatusConflict, "Conflicting meet name in session vs. query params.")
			return
		}
	}

	// if there's no conflict, update the session
	if meetNameParam != "" {
		session.Set("meetName", meetNameParam)
	}

	// if there's no conflict, update the session
	if posParam != "" {
		session.Set("desiredPosition", posParam)
	}

	// save the session
	if err := session.Save(); err != nil {
		logger.Error.Printf("[PerformLogin] Failed to save session: %v", err)
	}

	// then find the meetName + logo (if any), and render the login form
	rawMeetName := session.Get("meetName")
	var meetName, logo string
	if meetNameStr, ok := rawMeetName.(string); ok && meetNameStr != "" {
		meetName = meetNameStr
		creds := services.GetGlobalMeetCredentials()
		if creds == nil {
			logger.Error.Println("[PerformLogin] Global credentials not set")
			c.String(http.StatusInternalServerError, "Failed to load meet credentials")
			return
		}

		// find the matching meet
		for _, m := range creds.Meets {
			if m.Name == meetNameStr {
				logo = m.Logo
				break
			}
		}
	}

	//
	c.HTML(http.StatusOK, "login.html", gin.H{
		"MeetName": meetName,
		"Logo":     logo,
	})
}

// ------------------ login handling ------------------

// LoginHandler authenticates the user, prevents duplicate logins, and manages session storage.
func LoginHandler(c *gin.Context) {
	ctx := c.Request.Context()

	// create a subsegment
	ctx, seg := xray.BeginSubsegment(ctx, "LoginHandler")
	if seg != nil {
		defer seg.Close(nil)
	}

	// attach the updated context back to the request
	c.Request = c.Request.WithContext(ctx)

	session := sessions.Default(c)

	// retrieve meet name from session
	meetNameRaw := session.Get("meetName")
	meetName, ok := meetNameRaw.(string)
	if !ok || meetName == "" {
		logger.Warn.Println("[LoginHandler] No meet selected, redirecting to /choose-meet")
		c.Redirect(http.StatusFound, "/choose-meet")
		return
	}

	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || password == "" {
		logger.Warn.Println("[LoginHandler] Missing username or password")
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Please fill in all fields.",
		})
		return
	}

	// load meet credentials
	creds := services.GetGlobalMeetCredentials()
	if creds == nil {
		// handle the error or fallback
		logger.Error.Println("Global credentials not set")
		c.String(http.StatusInternalServerError, "Failed to load meet credentials")
		return
	}

	// check for superuser login
	if creds.Superuser != nil &&
		creds.Superuser.Username == username &&
		checkPasswordHash(password, creds.Superuser.Password) {
		session.Set("sudo", true)
		session.Set("isAdmin", true)
		session.Set("user", username)
		_ = session.Save()
		logger.Info.Printf("[LoginHandler] Superuser %s authenticated", username)
		c.Redirect(http.StatusFound, "/sudo")
		return
	}

	// validate the provided credentials against the selected meet
	var isAdmin bool
	var authenticated bool
	for _, m := range creds.Meets {
		if m.Name != meetName {
			continue
		}

		// primary admin
		if m.Admin.Username == username && checkPasswordHash(password, m.Admin.Password) {
			isAdmin = m.Admin.IsAdmin
			authenticated = true
			break
		}

		// secondary admins
		for _, sa := range m.SecondaryAdmins {
			if sa.Username == username && checkPasswordHash(password, sa.Password) {
				isAdmin = sa.IsAdmin
				authenticated = true
				break
			}
		}

		break // stop after checking this meet
	}

	if !authenticated {
		logger.Warn.Printf("[LoginHandler] Invalid login attempt for user=%s at meet=%s", username, meetName)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Invalid username or password.",
		})
		return
	}

	// prevent duplicate logins
	ActiveUsersMu.Lock()
	if ActiveUsers[username] {
		logger.Warn.Printf("[LoginHandler] User %s already logged in, denying second login", username)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Invalid username or password.",
			"Logo":     getLogoForMeet(meetName), // helper function
		})
		ActiveUsersMu.Unlock()
		return
	}
	ActiveUsers[username] = true
	ActiveUsersMu.Unlock()

	session.Set("user", username)
	session.Set("isAdmin", isAdmin)
	logger.Debug.Printf("[LoginHandler] Setting isAdmin=%v for user=%s", isAdmin, username)

	if err := session.Save(); err != nil {
		logger.Error.Printf("[LoginHandler] Failed to save session: %v", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Internal error, please try again.",
		})
		return
	}

	logger.Info.Printf("[LoginHandler] User %s authenticated for meet %s (isAdmin=%v)", username, meetName, isAdmin)

	// auto-claim desired position
	desiredPos := session.Get("desiredPosition")
	if desiredPos != nil {
		logger.Info.Printf("[LoginHandler] Attempting to auto-claim position=%s for user=%s", desiredPos, username)
		posString := desiredPos.(string)
		if err := occupancyService.SetPosition(meetName, posString, username); err != nil {
			logger.Warn.Printf("[LoginHandler] Auto-claim failed for user=%s on position=%s: %v", username, posString, err)
			c.String(http.StatusForbidden, "That seat is already taken or invalid. Please try another seat.")
			return
		}
		session.Set("refPosition", posString)
		_ = session.Save()

		switch posString {
		case "left":
			c.Redirect(http.StatusFound, "/left")
		case "center":
			c.Redirect(http.StatusFound, "/center")
		case "right":
			c.Redirect(http.StatusFound, "/right")
		default:
			c.Redirect(http.StatusFound, "/index")
		}
		return
	}

	// default redirect on success
	c.Redirect(http.StatusFound, "/index")
}

// helper function to retrieve logo for meet
func getLogoForMeet(meetName string) string {
	creds := services.GetGlobalMeetCredentials()
	if creds == nil {
		logger.Error.Println("[getLogoForMeet] Global credentials not set")
		// must return a string here
		return ""
	}
	for _, meet := range creds.Meets {
		if meet.Name == meetName {
			return meet.Logo
		}
	}
	return ""
}
