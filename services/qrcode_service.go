// Package services - services/qrcode_service.go
package services

import (
	"errors"

	"github.com/skip2/go-qrcode"
	"go-ref-lights/logger"
)

// QRCodeEncoder defines a function type for QR code generation
type QRCodeEncoder func(content string, level qrcode.RecoveryLevel, size int) ([]byte, error)

// GenerateQRCode creates a QR code using the provided encoder function
func GenerateQRCode(width, height int, encoder QRCodeEncoder) ([]byte, error) {
	if width <= 0 || height <= 0 {
		err := errors.New("invalid dimensions: width and height must be positive")
		logger.Error.Printf("QR code generation failed: %v", err)
		return nil, err
	}

	logger.Info.Printf("Generating QR code with dimensions %dx%d", width, height)
	data, err := encoder("https://referee-lights.michaelkingston.com.au/", qrcode.Medium, width)
	if err != nil {
		logger.Error.Printf("QR code generation failed: %v", err)
		return nil, err
	}

	logger.Info.Println("QR code generated successfully")
	return data, nil
}
