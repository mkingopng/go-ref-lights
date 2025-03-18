// Package services provides business logic for the application, including QR code generation.
// File: services/qrcode_service.go
package services

import (
	"errors"
	"github.com/skip2/go-qrcode"
	"go-ref-lights/logger"
)

// ------------ qr code encoding ------------

// QRCodeEncoder defines a function type for generating QR codes.
type QRCodeEncoder func(content string, level qrcode.RecoveryLevel, size int) ([]byte, error)

// Global encoder variable (defaults to qrcode.Encode); can be overridden in tests.
var encoder QRCodeEncoder = qrcode.Encode

// ------------ qr code generation ------------

// GenerateQRCode creates a QR code using the provided encoder function.
func GenerateQRCode(targetURL string, size int, level qrcode.RecoveryLevel) ([]byte, error) {
	// Validate input.
	if targetURL == "" {
		return nil, errors.New("cannot generate QR code: empty URL")
	}
	if size <= 0 {
		return nil, errors.New("invalid dimensions: width and height must be positive")
	}

	// Generate QR code using the (possibly overridden) encoder.
	pngBytes, err := encoder(targetURL, level, size)
	if err != nil {
		logger.Error.Printf("GenerateQRCode: Failed to create QR code for %s: %v", targetURL, err)
		return nil, err
	}

	return pngBytes, nil
}
