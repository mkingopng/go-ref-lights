// Package services provides business logic for the application, including QR code generation.
package services

import (
	"errors"

	"github.com/skip2/go-qrcode"
	"go-ref-lights/logger"
)

// define a function type for the encoder
type qrEncoderFunc func(content string, level qrcode.RecoveryLevel, size int) ([]byte, error)

// encoder is the function used to encode a QR code
// it defaults to qrcode.Encode, but can be overridden in tests
var encoder qrEncoderFunc = qrcode.Encode

// GenerateQRCode creates a QR code for the given URL.
// It returns a PNG as []byte, or an error.
func GenerateQRCode(targetURL string, size int, level qrcode.RecoveryLevel) ([]byte, error) {
	logger.Debug.Printf("[GenerateQRCode] Called with url=%s, size=%d, level=%v", targetURL, size, level)

	// basic validation
	if targetURL == "" {
		logger.Warn.Println("[GenerateQRCode] Empty targetURL provided")
		return nil, errors.New("cannot generate QR code: empty URL")
	}

	if size <= 0 {
		logger.Warn.Printf("[GenerateQRCode] Invalid size (%d). Must be > 0", size)
		return nil, errors.New("invalid dimensions: width and height must be positive")
	}

	logger.Debug.Printf("[GenerateQRCode] Invoking encoder for url=%s", targetURL)
	pngBytes, err := encoder(targetURL, level, size)
	if err != nil {
		logger.Error.Printf("[GenerateQRCode] Failed to create QR code for url=%s: %v", targetURL, err)
		return nil, err
	}

	logger.Info.Printf("[GenerateQRCode] QR code successfully generated for url=%s", targetURL)
	return pngBytes, nil
}
