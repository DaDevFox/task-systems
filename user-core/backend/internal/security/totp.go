package security

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/sirupsen/logrus"
)

const (
	// TOTPSecretLength is the length of the generated TOTP secret in bytes
	TOTPSecretLength = 20
	// DefaultBackupCodesCount is the number of backup codes generated
	DefaultBackupCodesCount = 10
	// BackupCodeLength is the length of each backup code
	BackupCodeLength = 8
	// TOTPValidityPeriod is the number of seconds a TOTP code is valid
	TOTPValidityPeriod = 30
	// TOTPAllowedSkew is the number of time steps to allow for clock skew
	TOTPAllowedSkew = 1
)

// TOTPSecret represents a TOTP secret with its metadata
type TOTPSecret struct {
	Secret      string
	AccountName string
	Issuer      string
}

// GenerateTOTPSecret generates a secure random TOTP secret
func GenerateTOTPSecret(accountName, issuer string, logger *logrus.Logger) (*TOTPSecret, error) {
	if accountName == "" {
		logger.Error("account name cannot be empty")
		return nil, fmt.Errorf("account name cannot be empty")
	}

	if issuer == "" {
		logger.Error("issuer cannot be empty")
		return nil, fmt.Errorf("issuer cannot be empty")
	}

	secret := make([]byte, TOTPSecretLength)
	_, err := rand.Read(secret)
	if err != nil {
		logger.WithError(err).Error("failed to generate random TOTP secret")
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	encodedSecret := base64.StdEncoding.EncodeToString(secret)
	totpSecret := &TOTPSecret{
		Secret:      encodedSecret,
		AccountName: accountName,
		Issuer:      issuer,
	}

	logger.WithField("account_name", accountName).Debug("TOTP secret generated successfully")
	return totpSecret, nil
}

// GenerateTOTPCode generates a TOTP code for a given secret at a specific time
func GenerateTOTPCode(secret string, t time.Time) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("secret cannot be empty")
	}

	code, err := totp.GenerateCodeCustom(secret, t, totp.ValidateOpts{
		Period:    TOTPValidityPeriod,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate TOTP code: %w", err)
	}

	return code, nil
}

// VerifyTOTPCode validates a TOTP code against a secret
func VerifyTOTPCode(secret, code string, logger *logrus.Logger) (bool, error) {
	if secret == "" {
		logger.Error("TOTP secret cannot be empty")
		return false, fmt.Errorf("TOTP secret cannot be empty")
	}

	if code == "" {
		logger.Error("TOTP code cannot be empty")
		return false, fmt.Errorf("TOTP code cannot be empty")
	}

	valid, err := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    TOTPValidityPeriod,
		Skew:      TOTPAllowedSkew,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})

	if err != nil {
		logger.WithError(err).Warn("TOTP code verification failed")
		return false, fmt.Errorf("TOTP verification error: %w", err)
	}

	logger.Debug("TOTP code verification completed")
	return valid, nil
}

// GenerateQRCode generates a QR code for TOTP setup
func GenerateQRCode(totpSecret *TOTPSecret, size int, logger *logrus.Logger) ([]byte, error) {
	if totpSecret == nil {
		logger.Error("TOTP secret cannot be nil")
		return nil, fmt.Errorf("TOTP secret cannot be nil")
	}

	if size <= 0 {
		size = 200 // Default QR code size
	}

	key, err := otp.NewKeyFromURL(
		fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
			urlEncode(totpSecret.Issuer),
			urlEncode(totpSecret.AccountName),
			totpSecret.Secret,
			urlEncode(totpSecret.Issuer),
		),
	)

	if err != nil {
		logger.WithError(err).Error("failed to create TOTP key from URL")
		return nil, fmt.Errorf("failed to create TOTP key: %w", err)
	}

	// Generate QR code
	qrCode, err := key.Image(size)
	if err != nil {
		logger.WithError(err).Error("failed to generate QR code image")
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	logger.Debug("QR code generated successfully")
	return qrCode, nil
}

// GenerateBackupCodes generates random backup codes for TOTP recovery
func GenerateBackupCodes(count int, logger *logrus.Logger) ([]string, error) {
	if count <= 0 {
		count = DefaultBackupCodesCount
	}

	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code, err := generateSecureRandomCode(BackupCodeLength)
		if err != nil {
			logger.WithError(err).Error("failed to generate backup code")
			return nil, fmt.Errorf("failed to generate backup code: %w", err)
		}
		codes[i] = code
	}

	logger.WithField("count", count).Debug("backup codes generated successfully")
	return codes, nil
}

// generateSecureRandomCode generates a secure random code
func generateSecureRandomCode(length int) (string, error) {
	// Generate enough bytes to produce the requested length in base32
	bytesNeeded := (length*5 + 7) / 8
	bytes := make([]byte, bytesNeeded)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Encode to base32 for readability (uppercase, no padding)
	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	encoded := encoder.EncodeToString(bytes)

	// Return only the requested length
	if len(encoded) >= length {
		return encoded[:length], nil
	}

	// If for some reason it's shorter, pad with zeros
	for len(encoded) < length {
		encoded += "A"
	}
	return encoded, nil
}

// urlEncode performs URL encoding for QR code data
func urlEncode(s string) string {
	s = strings.ReplaceAll(s, " ", "%20")
	s = strings.ReplaceAll(s, ":", "%3A")
	s = strings.ReplaceAll(s, "@", "%40")
	return s
}
