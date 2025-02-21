// services/qrcode_service.go
package services

import (
	"errors"
	"github.com/skip2/go-qrcode"
)

// QRCodeEncoder defines a function type for QR code generation
type QRCodeEncoder func(content string, level qrcode.RecoveryLevel, size int) ([]byte, error)

// GenerateQRCode creates a QR code using the provided encoder function
func GenerateQRCode(width, height int, encoder QRCodeEncoder) ([]byte, error) {
	if width <= 0 || height <= 0 {
		return nil, errors.New("invalid dimensions: width and height must be positive")
	}

	return encoder("http://your-url-here", qrcode.Medium, width)
}
