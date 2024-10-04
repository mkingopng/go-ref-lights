// services/qrcode_service.go
package services

import (
	"github.com/skip2/go-qrcode"
)

// GenerateQRCode creates a QR code for the given dimensions
func GenerateQRCode(width, height int) ([]byte, error) {
	// Replace with your application URL or dynamic content
	applicationURL := "http://localhost:8080"

	// Generate QR code
	png, err := qrcode.Encode(applicationURL, qrcode.Medium, width)
	if err != nil {
		return nil, err
	}
	return png, nil
}
