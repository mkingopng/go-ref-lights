// Package middleware provides request filters and security checks for the application.
// File: middleware/auth.go
package middleware

import (
	"net/http"

	"go-ref-lights/logger"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// -------------- authentication middleware --------------

// AuthRequired is a middleware that ensures the user is logged in.
/*How it works:
- Retrieves the session from the request context.
- Checks if the "user" session variable is set.
- If no user is found, redirects to "/choose-meet" and aborts execution.
- Otherwise, the request proceeds.*/
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	clientIP := c.ClientIP()
	user := session.Get("user")

	// Create authentication context for structured logging
	authContext := logger.NewAuthenticationContext("session_check", "", clientIP)
	authContext["userAgent"] = c.Request.UserAgent()
	authContext["path"] = c.Request.URL.Path
	authContext["method"] = c.Request.Method
	authContext["hasUser"] = user != nil

	if user != nil {
		if userStr, ok := user.(string); ok {
			authContext["username"] = userStr
		}
	}

	logger.LogDebugWithContext(authContext, "Checking user session authentication")

	if user == nil {
		authContext["action"] = "redirect_unauthorized"
		authContext["redirectTo"] = "/choose-meet"
		logger.LogWarnWithContext(authContext, "Unauthorized access attempt - no user session found, redirecting to /choose-meet")
		c.Redirect(http.StatusFound, "/choose-meet")
		c.Abort()
		return
	}

	authContext["action"] = "authorized"
	logger.LogDebugWithContext(authContext, "User session authenticated successfully")
	c.Next()
}

// AdminRequired is a middleware that restricts access to admin-only routes.
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		clientIP := c.ClientIP()
		isAdmin, ok := session.Get("isAdmin").(bool)

		// Create authorization context for structured logging
		authzContext := logger.NewAuthenticationContext("admin_check", "", clientIP)
		authzContext["userAgent"] = c.Request.UserAgent()
		authzContext["path"] = c.Request.URL.Path
		authzContext["method"] = c.Request.Method
		authzContext["isAdmin"] = isAdmin
		authzContext["adminFlagPresent"] = ok
		authzContext["requiredRole"] = "admin"

		if user := session.Get("user"); user != nil {
			if userStr, ok := user.(string); ok {
				authzContext["username"] = userStr
			}
		}

		logger.LogDebugWithContext(authzContext, "Checking admin authorization")

		if !ok || !isAdmin {
			authzContext["action"] = "access_denied"
			authzContext["reason"] = "insufficient_privileges"
			logger.LogWarnWithContext(authzContext, "Admin access denied - insufficient privileges")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Admin privileges required"})
			return
		}

		authzContext["action"] = "access_granted"
		logger.LogDebugWithContext(authzContext, "Admin access granted")
		c.Next()
	}
}

// SudoRequired ensures the user has superuser (sudo) privileges.
func SudoRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		clientIP := c.ClientIP()
		isSudo, ok := session.Get("sudo").(bool)

		// Create sudo authorization context for structured logging
		sudoContext := logger.NewAuthenticationContext("sudo_check", "", clientIP)
		sudoContext["userAgent"] = c.Request.UserAgent()
		sudoContext["path"] = c.Request.URL.Path
		sudoContext["method"] = c.Request.Method
		sudoContext["isSudo"] = isSudo
		sudoContext["sudoFlagPresent"] = ok
		sudoContext["requiredRole"] = "superuser"

		if user := session.Get("user"); user != nil {
			if userStr, ok := user.(string); ok {
				sudoContext["username"] = userStr
			}
		}

		logger.LogDebugWithContext(sudoContext, "Checking superuser authorization")

		if !ok || !isSudo {
			sudoContext["action"] = "access_denied"
			sudoContext["reason"] = "insufficient_privileges"
			logger.LogWarnWithContext(sudoContext, "Superuser access denied - insufficient privileges")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Superuser privileges required"})
			c.Abort()
			return
		}

		sudoContext["action"] = "access_granted"
		logger.LogDebugWithContext(sudoContext, "Superuser access granted")
		c.Next()
	}
}

// MeetRequired ensures a valid meetName is present in the session.
// Redirects to / if missing.
func MeetRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		meetName, ok := session.Get("meetName").(string)
		clientIP := c.ClientIP()

		// Create meet validation context for structured logging
		meetContext := logger.NewHTTPContext(c.Request.Method, c.Request.URL.Path, c.Request.UserAgent(), clientIP, 0)
		meetContext["component"] = "middleware"
		meetContext["action"] = "meet_validation"
		meetContext["meetNamePresent"] = ok
		meetContext["meetNameEmpty"] = meetName == ""

		if ok && meetName != "" {
			meetContext["meetName"] = meetName
		}

		if user := session.Get("user"); user != nil {
			if userStr, ok := user.(string); ok {
				meetContext["username"] = userStr
			}
		}

		logger.LogDebugWithContext(meetContext, "Validating meet name in session")

		if !ok || meetName == "" {
			meetContext["action"] = "redirect_no_meet"
			meetContext["redirectTo"] = "/"
			logger.LogWarnWithContext(meetContext, "Missing or empty meetName in session - redirecting to meet selection")
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}

		meetContext["action"] = "meet_validated"
		logger.LogDebugWithContext(meetContext, "Meet name validation successful: %s", meetName)
		c.Next()
	}
}
