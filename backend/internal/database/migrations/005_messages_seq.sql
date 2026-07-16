ALTER TABLE messages ADD COLUMN IF NOT EXISTS seq BIGSERIAL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_messages_seq ON messages (seq);
CREATE INDEX IF NOT EXISTS idx_messages_chat_seq ON messages (chat_id, seq);
