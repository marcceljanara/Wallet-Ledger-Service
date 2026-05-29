package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomAlphanumeric(n int) (string, error) {
	b := make([]byte, n)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random index: %w", err)
		}
		b[i] = charset[num.Int64()]
	}
	return string(b), nil
}

func GenerateWalletID() (string, error) {
	randStr, err := randomAlphanumeric(8)
	if err != nil {
		return "", err
	}
	return "WLT-" + randStr, nil
}

func GenerateTransactionRef() (string, error) {
	randStr, err := randomAlphanumeric(8)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("TXN-%s-%s", time.Now().Format("20060102"), randStr), nil
}

func GenerateLedgerEntryID() (string, error) {
	randStr, err := randomAlphanumeric(10)
	if err != nil {
		return "", err
	}
	return "LED-" + randStr, nil
}

func GenerateAuditLogID() (string, error) {
	randStr, err := randomAlphanumeric(10)
	if err != nil {
		return "", err
	}
	return "AUD-" + randStr, nil
}
