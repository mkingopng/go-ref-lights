// Package services provides business logic for the application, including QR code generation.
package services

import (
	"errors"

	"github.com/skip2/go-qrcode"
	"go-ref-lights/logger"
)

// Define a function type for the encoder.
type qrEncoderFunc func(content string, level qrcode.RecoveryLevel, size int) ([]byte, error)

// encoder is the function used to encode a QR code.
// It defaults to qrcode.Encode, but can be overridden in tests.
var encoder qrEncoderFunc = qrcode.Encode

// GenerateQRCode creates a QR code for the given URL.
// It returns a PNG as []byte, or an error.
func GenerateQRCode(targetURL string, size int, level qrcode.RecoveryLevel) ([]byte, error) {
	// Basic validation
	if targetURL == "" {
		return nil, errors.New("cannot generate QR code: empty URL")
	}
	if size <= 0 {
		return nil, errors.New("invalid dimensions: width and height must be positive")
	}

	pngBytes, err := encoder(targetURL, level, size)
	if err != nil {
		logger.Error.Printf("GenerateQRCode: Failed to create QR code for %s: %v", targetURL, err)
		return nil, err
	}

	return pngBytes, nil
}
