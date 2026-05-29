package utils

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateWalletID(t *testing.T) {
	id := GenerateWalletID()
	assert.Len(t, id, 12)
	assert.Regexp(t, regexp.MustCompile(`^WLT-[A-Z0-9]{8}$`), id)

	id2 := GenerateWalletID()
	assert.NotEqual(t, id, id2)
}

func TestGenerateTransactionRef(t *testing.T) {
	ref := GenerateTransactionRef()
	assert.Len(t, ref, 21)
	assert.Regexp(t, regexp.MustCompile(`^TXN-\d{8}-[A-Z0-9]{8}$`), ref)
}

func TestGenerateLedgerEntryID(t *testing.T) {
	id := GenerateLedgerEntryID()
	assert.Len(t, id, 14)
	assert.Regexp(t, regexp.MustCompile(`^LED-[A-Z0-9]{10}$`), id)
}

func TestGenerateAuditLogID(t *testing.T) {
	id := GenerateAuditLogID()
	assert.Len(t, id, 14)
	assert.Regexp(t, regexp.MustCompile(`^AUD-[A-Z0-9]{10}$`), id)
}
