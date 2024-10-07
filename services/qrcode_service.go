// services/qrcode_service.go
package services

import (
	"github.com/skip2/go-qrcode"
	"github.com/spf13/viper"
)

// GenerateQRCode creates a QR code for the given dimensions
func GenerateQRCode(width, height int) ([]byte, error) {
	applicationURL := viper.GetString("application_url")
	if applicationURL == "" {
		applicationURL = ""
	}

	png, err := qrcode.Encode(applicationURL, qrcode.Medium, width)
	if err != nil {
		return nil, err
	}
	return png, nil
}
