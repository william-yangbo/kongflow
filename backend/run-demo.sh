#!/bin/bash

echo "=== Starting PostgreSQL for Demo ==="
docker run --name demo-postgres -e POSTGRES_DB=kongflow_dev -e POSTGRES_USER=kong -e POSTGRES_PASSWORD=flow2025 -p 5432:5432 -d postgres:15-alpine

echo "Waiting for PostgreSQL to be ready..."
sleep 8

echo "Creating table..."
docker exec demo-postgres psql -U kong -d kongflow_dev -c "
CREATE TABLE IF NOT EXISTS secret_store (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS \$\$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
\$\$ language 'plpgsql';

CREATE TRIGGER update_secret_store_updated_at BEFORE UPDATE
    ON secret_store FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();"

echo "Running demo..."
go run cmd/demo/main.go

echo "Cleaning up..."
docker stop demo-postgres
docker rm demo-postgres