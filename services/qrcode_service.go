// services/qrcode_service.go
package services

import (
	"github.com/skip2/go-qrcode"
)

// GenerateQRCode creates a QR code for the given dimensions
func GenerateQRCode(width, height int) ([]byte, error) {
	qr, err := qrcode.Encode("http://your-url-here", qrcode.Medium, width)
	if err != nil {
		return nil, err
	}
	return qr, nil
}
