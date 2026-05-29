CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_no VARCHAR(30) NOT NULL UNIQUE,
    type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    amount DECIMAL(18,2) NOT NULL CHECK (amount > 0),
    source_wallet_id VARCHAR(20) REFERENCES wallets(id),
    target_wallet_id VARCHAR(20) REFERENCES wallets(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_source_wallet ON transactions(source_wallet_id);
CREATE INDEX idx_transactions_target_wallet ON transactions(target_wallet_id);
CREATE INDEX idx_transactions_reference_no ON transactions(reference_no);
CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);
