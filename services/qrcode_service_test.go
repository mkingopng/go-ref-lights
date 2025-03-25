//go:build unit
// +build unit

// file: services/qrcode_service_test.go
//
package services

import (
	"errors"
	"testing"

	"github.com/skip2/go-qrcode"
	"github.com/stretchr/testify/assert"
	"go-ref-lights/websocket"
)

// mock encoder function (successful)
func mockQRCodeEncoderSuccess(content string, level qrcode.RecoveryLevel, size int) ([]byte, error) {
	return []byte("mock_qr_code_data"), nil
}

// mock encoder function (failure)
func mockQRCodeEncoderFailure(content string, level qrcode.RecoveryLevel, size int) ([]byte, error) {
	return nil, errors.New("QR code generation failed")
}

// TestGenerateQRCode_Success: Test QR Code Generation Success
func TestGenerateQRCode_Success(t *testing.T) {
	websocket.InitTest()

	// Optionally, you could override encoder here if needed:
	// originalEncoder := encoder
	// defer func() { encoder = originalEncoder }()
	// encoder = mockQRCodeEncoderSuccess

	// ensure correct argument order: content, size, recoveryLevel
	data, err := GenerateQRCode("https://example.com", 200, qrcode.Medium)

	assert.NoError(t, err)
	assert.NotNil(t, data)
}

// TestGenerateQRCode_InvalidURL: Test QR Code Generation Fails Due to Invalid URL
func TestGenerateQRCode_InvalidDimensions(t *testing.T) {
	websocket.InitTest()

	data, err := GenerateQRCode("https://example.com", -100, qrcode.Medium)

	// expect error for invalid dimensions
	assert.Error(t, err, "Expected an error for negative dimensions")
	assert.Nil(t, data, "Data should be nil for invalid dimensions")
	assert.Equal(t, "invalid dimensions: width and height must be positive", err.Error())
}

// TestGenerateQRCode_InvalidURL: Test QR Code Generation Fails Due to Invalid URL
func TestGenerateQRCode_EncoderFails(t *testing.T) {
	websocket.InitTest()

	// override the encoder to simulate failure.
	originalEncoder := encoder
	defer func() { encoder = originalEncoder }()
	encoder = mockQRCodeEncoderFailure

	data, err := GenerateQRCode("https://example.com", 200, qrcode.High)

	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Equal(t, "QR code generation failed", err.Error())
}
