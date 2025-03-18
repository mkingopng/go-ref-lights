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

// ------------ qr code generation ------------

// GenerateQRCode creates a QR code using the provided encoder function
func GenerateQRCode(targetURL string, size int, level qrcode.RecoveryLevel) ([]byte, error) {
	// Basic validation
	if targetURL == "" {
		return nil, errors.New("cannot generate QR code: empty URL")
	}
	if size <= 0 {
		size = 300 // default fallback
	}

	pngBytes, err := qrcode.Encode(targetURL, level, size)
	if err != nil {
		logger.Error.Printf("GenerateQRCode: Failed to create QR code for %s: %v", targetURL, err)
		return nil, err
	}

	return pngBytes, nil
}
