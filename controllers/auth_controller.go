// Package controllers provides authentication and session management for users.
// File: controllers/auth_controller.go
package controllers

import (
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

var occupancyService services.OccupancyServiceInterface

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
func ComparePasswords(username, hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	if err != nil {
		logger.Warn.Printf("[ComparePasswords] Password mismatch for user=%s", username)
	}
	return err == nil
}

// SetMeetHandler sets the selected meet in the session and redirects to the meet page.
func SetMeetHandler(c *gin.Context) {
	meetName := c.PostForm("meetName")
	if meetName == "" {
		logger.Warn.Printf("[SetMeetHandler] No meetName provided in form POST from %s", c.ClientIP())
		c.HTML(http.StatusBadRequest, "choose-meet.html", gin.H{"Error": "Please select a meet."})
		return
	}

	session := sessions.Default(c)
	session.Set("meetName", meetName)

	if err := session.Save(); err != nil {
		logger.Error.Printf("[SetMeetHandler] Failed to save session for meet=%s from %s: %v", meetName, c.ClientIP(), err)
		c.HTML(http.StatusInternalServerError, "choose-meet.html", gin.H{"Error": "Internal error, please try again."})
		return
	}

	logger.Info.Printf("[SetMeetHandler] Meet selected: %s by client %s — redirecting to login", meetName, c.ClientIP())
	c.Redirect(http.StatusFound, "/login")
}

// ----------------------- meet selection ----------------------------------

// MeetHandler retrieves the meet details from session and renders the home page with the appropriate logo.
func MeetHandler(c *gin.Context) {
	session := sessions.Default(c)
	storedMeet := session.Get("meetName")
	if storedMeet == nil {
		logger.Warn.Printf("[MeetHandler] No meetName in session from %s", c.ClientIP())
		c.HTML(http.StatusBadRequest, "choose-meet.html", gin.H{"Error": "No meet selected."})
		return
	}
	meetName := storedMeet.(string)

	// load meet credentials using the injectable function.
	creds := services.GetGlobalMeetCredentials()
	if creds == nil {
		logger.Error.Println("[MeetHandler] Global meet credentials were nil — possible misconfiguration or init failure")
		c.HTML(http.StatusInternalServerError, "choose-meet.html", gin.H{"Error": "Internal error loading meets."})
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
		logger.Warn.Printf("[MeetHandler] Unknown meetName=%s from %s — not found in loaded credentials", meetName, c.ClientIP())
		c.HTML(http.StatusNotFound, "choose-meet.html", gin.H{"Error": "Meet not found."})
		return
	}

	// render the template with the correct logo.
	logger.Info.Printf("[MeetHandler] Rendering index for meet=%s, IP=%s", currentMeet.Name, c.ClientIP())
	c.HTML(http.StatusOK, "index.html", gin.H{
		"meetName": currentMeet.Name,
		"logo":     currentMeet.Logo,
	})
}

// ----------------------- admin actions -----------------------------------

// ForceLogoutHandler forcibly logs out a user (admin action).
func ForceLogoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	isAdmin := session.Get("isAdmin")

	if isAdmin == nil || isAdmin != true {
		logger.Warn.Println("[ForceLogoutHandler] Unauthorized attempt to force logout a user.")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin privileges required"})
		return
	}

	username := c.PostForm("username")
	if username == "" {
		logger.Warn.Println("[ForceLogoutHandler] Missing username in POST body.")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing username parameter"})
		return
	}

	ActiveUsersMu.Lock()
	defer ActiveUsersMu.Unlock()

	if _, exists := ActiveUsers[username]; !exists {
		logger.Warn.Printf("[ForceLogoutHandler] Attempted to log out non-existent user: %s", username)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not logged in"})
		return
	}

	delete(ActiveUsers, username)
	logger.Info.Printf("[ForceLogoutHandler] Admin forcibly logged out user: %s", username)
	c.JSON(http.StatusOK, gin.H{"message": "User logged out successfully"})
}

// --------------------- active user tracking ------------------------------

// ActiveUsersHandler returns a list of currently active users (admin action).
func ActiveUsersHandler(c *gin.Context) {
	session := sessions.Default(c)
	isAdmin := session.Get("isAdmin")

	if isAdmin == nil || isAdmin != true {
		logger.Warn.Printf("[ActiveUsersHandler] Unauthorized attempt to access active user list from %s", c.ClientIP())
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

	logger.Info.Printf("[ActiveUsersHandler] Admin at %s requested active users: count=%d", c.ClientIP(), len(userList))
	c.JSON(http.StatusOK, gin.H{"users": userList})
}

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

	// render the login page
	c.HTML(http.StatusOK, "login.html", gin.H{
		"MeetName": meetName,
		"Logo":     logo,
	})
}

// ------------------ login handling ------------------

// LoginHandler authenticates the user, prevents duplicate logins, and manages session storage.
func LoginHandler(c *gin.Context) {
	session := sessions.Default(c)

	// retrieve meet name from session
	meetNameRaw := session.Get("meetName")
	meetName, ok := meetNameRaw.(string)
	if !ok || meetName == "" {
		logger.Warn.Printf("[LoginHandler] No meetName in session — redirecting. IP=%s", c.ClientIP())
		c.Redirect(http.StatusFound, "/choose-meet")
		return
	}

	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || password == "" {
		logger.Warn.Printf("[LoginHandler] Missing login fields for meet=%s from %s", meetName, c.ClientIP())
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Please fill in all fields.",
		})
		return
	}

	// load meet credentials
	creds := services.GetGlobalMeetCredentials()
	if creds == nil {
		logger.Error.Printf("[LoginHandler] Global credentials nil — meet=%s, IP=%s", meetName, c.ClientIP())
		c.String(http.StatusInternalServerError, "Failed to load meet credentials")
		return
	}

	// 1) Check for top-level superuser (unchanged)
	if creds.Superuser != nil &&
		creds.Superuser.Username == username &&
		checkPasswordHash(password, creds.Superuser.Password) {
		session.Set("sudo", true)
		session.Set("isAdmin", true)
		session.Set("user", username)
		_ = session.Save()
		c.Redirect(http.StatusFound, "/sudo")
		return
	}

	// 2) Check for a meet-level admin or secondaryAdmin, including 'sudo' field
	var isAdmin, isSudo, authenticated bool

	for _, m := range creds.Meets {
		if m.Name != meetName {
			continue
		}

		// primary admin
		if m.Admin.Username == username && checkPasswordHash(password, m.Admin.Password) {
			isAdmin = m.Admin.IsAdmin
			isSudo = m.Admin.Sudo
			authenticated = true
			break
		}

		// secondary admins
		for _, sa := range m.SecondaryAdmins {
			if sa.Username == username && checkPasswordHash(password, sa.Password) {
				isAdmin = sa.IsAdmin
				isSudo = sa.Sudo
				authenticated = true
				break
			}
		}

		break // break after we processed this meet
	}

	// if still not authenticated, fail
	if !authenticated {
		logger.Warn.Printf("[LoginHandler] Invalid login for user=%s at meet=%s from %s", username, meetName, c.ClientIP())
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Invalid username or password.",
		})
		return
	}

	// 3) Prevent duplicate logins
	ActiveUsersMu.Lock()
	if ActiveUsers[username] {
		logger.Warn.Printf("[LoginHandler] Duplicate login attempt denied for user=%s at meet=%s from %s", username, meetName, c.ClientIP())
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "You are already logged in elsewhere. Force logout?",
			"Username": username,
			"Logo":     getLogoForMeet(meetName),
		})
		ActiveUsersMu.Unlock()
		return
	}
	ActiveUsers[username] = true
	ActiveUsersMu.Unlock()

	// 4) Store user info in session (ADDED: store isSudo)
	session.Set("user", username)
	session.Set("isAdmin", isAdmin)
	session.Set("sudo", isSudo)

	if err := session.Save(); err != nil {
		logger.Error.Printf("[LoginHandler] Session save failed for user=%s at meet=%s from %s: %v", username, meetName, c.ClientIP(), err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Internal error, please try again.",
		})
		return
	}

	logger.Info.Printf("[LoginHandler] Authenticated user=%s at meet=%s (isAdmin=%v, isSudo=%v) from %s", username, meetName, isAdmin, isSudo, c.ClientIP())

	// 5) If a position was desired, auto-claim it
	desiredPos := session.Get("desiredPosition")
	if desiredPos != nil {
		posString := desiredPos.(string)
		logger.Info.Printf("[LoginHandler] Auto-claim requested: user=%s position=%s meet=%s", username, posString, meetName)
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
			return
		case "center":
			c.Redirect(http.StatusFound, "/center")
			return
		case "right":
			c.Redirect(http.StatusFound, "/right")
			return
		default:
			// fallback
			c.Redirect(http.StatusFound, "/index")
			return
		}
	}

	// 6) Otherwise, just send them to /index
	c.Redirect(http.StatusFound, "/index")
}

// getLogoForMeet returns the logo URL or path for a given meetName.
// Logs a warning if the meet is not found.
func getLogoForMeet(meetName string) string {
	creds := services.GetGlobalMeetCredentials()
	if creds == nil {
		logger.Error.Printf("[getLogoForMeet] Global credentials not set while fetching logo for meet=%s", meetName)
		return ""
	}
	for _, meet := range creds.Meets {
		if meet.Name == meetName {
			return meet.Logo
		}
	}
	logger.Warn.Printf("[getLogoForMeet] No match found for meet=%s — returning empty logo", meetName)
	return ""
}

// ForceMyLogin forcibly removes the user from ActiveUsers and redirects back to login.
func ForceMyLogin(c *gin.Context) {
	username := c.PostForm("username")
	if username == "" {
		logger.Warn.Printf("[ForceMyLogin] Missing username in POST from %s", c.ClientIP())
		c.HTML(http.StatusBadRequest, "login.html", gin.H{"Error": "Missing username."})
		return
	}

	// remove from ActiveUsers
	ActiveUsersMu.Lock()
	delete(ActiveUsers, username)
	ActiveUsersMu.Unlock()

	logger.Info.Printf("[ForceMyLogin] User %s forcibly removed from ActiveUsers by client %s", username, c.ClientIP())

	// redirect to login
	c.Redirect(http.StatusFound, "/login?username="+username)
}
