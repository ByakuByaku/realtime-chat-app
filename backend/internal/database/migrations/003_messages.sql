--таблицы
CREATE TABLE IF NOT EXISTS messages (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id       UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    sender_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    body          TEXT NOT NULL,
    client_msg_id TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (chat_id, client_msg_id)
);
--триггеры
CREATE OR REPLACE FUNCTION check_sender_is_member()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.sender_id IS NULL THEN
        RETURN NEW;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM chat_members
        WHERE chat_id = NEW.chat_id AND user_id = NEW.sender_id
    ) THEN
        RAISE EXCEPTION 'Пользователь % не состоит в чате %', NEW.sender_id, NEW.chat_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_check_sender_is_member ON messages;
CREATE TRIGGER trg_check_sender_is_member
BEFORE INSERT ON messages
FOR EACH ROW
EXECUTE FUNCTION check_sender_is_member();

CREATE OR REPLACE FUNCTION update_chat_last_message()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE chats SET last_message_at = NEW.created_at WHERE id = NEW.chat_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_update_chat_last_message ON messages;
CREATE TRIGGER trg_update_chat_last_message
AFTER INSERT ON messages
FOR EACH ROW
EXECUTE FUNCTION update_chat_last_message();
