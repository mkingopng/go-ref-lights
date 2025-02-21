// file: services/qrcode_service_test.go
package services

import (
	"bytes"
	"errors"
	"image/png"
	"testing"

	"github.com/skip2/go-qrcode"
	"github.com/stretchr/testify/assert"
)

// Test: QR code generation should return non-empty data
func TestGenerateQRCode_Success(t *testing.T) {
	encoder := func(content string, level qrcode.RecoveryLevel, size int) ([]byte, error) {
		return qrcode.Encode(content, level, size)
	}

	data, err := GenerateQRCode(250, 250, encoder)

	// Ensure no error is returned
	assert.NoError(t, err, "Expected no error when generating QR code")

	// Ensure QR code data is not empty
	assert.NotEmpty(t, data, "QR code data should not be empty")

	// Ensure the output is a valid PNG
	_, pngErr := png.Decode(bytes.NewReader(data))
	assert.NoError(t, pngErr, "Expected valid PNG format")
}

// Test: Invalid QR code dimensions should return an error
func TestGenerateQRCode_InvalidSize(t *testing.T) {
	encoder := func(content string, level qrcode.RecoveryLevel, size int) ([]byte, error) {
		return qrcode.Encode(content, level, size)
	}

	data, err := GenerateQRCode(0, 0, encoder)

	// Ensure error is returned
	assert.Error(t, err, "Expected an error for invalid dimensions")

	// Ensure QR code data is empty
	assert.Empty(t, data, "QR code data should be empty on failure")
}

// Mock encoder function to simulate an internal failure
func TestGenerateQRCode_EncodeFailure(t *testing.T) {
	mockEncoder := func(content string, level qrcode.RecoveryLevel, size int) ([]byte, error) {
		return nil, errors.New("simulated QR code generation failure")
	}

	data, err := GenerateQRCode(250, 250, mockEncoder)

	// Ensure error is returned
	assert.Error(t, err, "Expected an error for QR code generation failure")

	// Ensure QR code data is empty
	assert.Empty(t, data, "QR code data should be empty on failure")
}
