CREATE TABLE ledger_entries (
    id VARCHAR(20) PRIMARY KEY,
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    wallet_id VARCHAR(20) NOT NULL REFERENCES wallets(id),
    entry_type VARCHAR(10) NOT NULL,
    amount DECIMAL(18,2) NOT NULL CHECK (amount > 0),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ledger_entries_transaction_id ON ledger_entries(transaction_id);
CREATE INDEX idx_ledger_entries_wallet_id ON ledger_entries(wallet_id);
