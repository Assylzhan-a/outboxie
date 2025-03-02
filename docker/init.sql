-- Create outbox table
CREATE TABLE IF NOT EXISTS outbox_messages (
    id UUID PRIMARY KEY,
    topic VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    retry_count INT NOT NULL DEFAULT 0,
    error TEXT,
    sequence_number BIGSERIAL NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_outbox_messages_status ON outbox_messages(status);
CREATE INDEX IF NOT EXISTS idx_outbox_messages_created_at ON outbox_messages(created_at);
CREATE INDEX IF NOT EXISTS idx_outbox_messages_sequence_number ON outbox_messages(sequence_number);

-- Create leader election table
CREATE TABLE IF NOT EXISTS leader_election (
    id TEXT PRIMARY KEY,
    instance_id TEXT NOT NULL,
    last_heartbeat TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create orders table for the example
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
); 