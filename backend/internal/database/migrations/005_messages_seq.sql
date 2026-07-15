ALTER TABLE messages ADD COLUMN seq BIGSERIAL;

CREATE UNIQUE INDEX ux_messages_seq ON messages (seq);
CREATE INDEX idx_messages_chat_seq ON messages (chat_id, seq);
