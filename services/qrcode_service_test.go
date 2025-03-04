// file: services/qrcode_service_test.go
package services

import (
	"errors"
	"testing"

	"github.com/skip2/go-qrcode"
	"github.com/stretchr/testify/assert"
)

// Mock encoder function (successful)
func mockQRCodeEncoderSuccess(content string, level qrcode.RecoveryLevel, size int) ([]byte, error) {
	return []byte("mock_qr_code_data"), nil
}

// Mock encoder function (failure)
func mockQRCodeEncoderFailure(content string, level qrcode.RecoveryLevel, size int) ([]byte, error) {
	return nil, errors.New("QR code generation failed")
}

// Test: Generate QR Code Successfully
func TestGenerateQRCode_Success(t *testing.T) {
	data, err := GenerateQRCode(200, 200, mockQRCodeEncoderSuccess)

	assert.NoError(t, err)
	assert.NotNil(t, data)
	assert.Equal(t, "mock_qr_code_data", string(data))
}

// Test: Fail QR Code Generation Due to Negative Dimensions
func TestGenerateQRCode_InvalidDimensions(t *testing.T) {
	data, err := GenerateQRCode(-100, 200, mockQRCodeEncoderSuccess)

	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Equal(t, "invalid dimensions: width and height must be positive", err.Error())
}

// Test: QR Code Generation Fails Due to Encoder Error
func TestGenerateQRCode_EncoderFails(t *testing.T) {
	data, err := GenerateQRCode(200, 200, mockQRCodeEncoderFailure)

	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Equal(t, "QR code generation failed", err.Error())
}
