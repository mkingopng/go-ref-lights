// services/qrcode_service.go
package services

import (
	"github.com/skip2/go-qrcode"
	"os"
)

// GenerateQRCode creates a QR code for the given dimensions
func GenerateQRCode(width, height int) ([]byte, error) {
	applicationURL := os.Getenv("APPLICATION_URL")
	if applicationURL == "" {
		applicationURL = "http://localhost:8080" // Default for local testing
	}

	png, err := qrcode.Encode(applicationURL, qrcode.Medium, width)
	if err != nil {
		return nil, err
	}
	return png, nil
}
