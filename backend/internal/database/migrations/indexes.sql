CREATE INDEX IF NOT EXISTS idx_messages_chat_created_at ON messages (chat_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_chat_members_user_id ON chat_members (user_id);

CREATE UNIQUE INDEX IF NOT EXISTS ux_refresh_tokens_token_hash ON refresh_tokens (token_hash);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens (user_id) WHERE revoked = false;