-- CreateTable SecretStore
CREATE TABLE secret_store (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create index for faster lookups
CREATE INDEX idx_secret_store_created_at ON secret_store(created_at);
CREATE INDEX idx_secret_store_updated_at ON secret_store(updated_at);

-- Create function to update updated_at automatically
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger to auto-update updated_at
CREATE TRIGGER update_secret_store_updated_at
    BEFORE UPDATE ON secret_store
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();