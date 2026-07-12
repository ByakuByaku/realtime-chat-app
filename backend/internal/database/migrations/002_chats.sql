--таблицы
CREATE TABLE chats (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type            TEXT NOT NULL CHECK (type IN ('direct', 'group')),
    title           TEXT,
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    direct_key      TEXT,
    last_message_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
 
CREATE TABLE chat_members (
    chat_id   UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role      TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('member', 'admin')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (chat_id, user_id)
);

--триггеры
CREATE OR REPLACE FUNCTION set_chat_direct_key()
RETURNS TRIGGER AS $$
DECLARE
    v_type     TEXT;
    v_chat_id  UUID;
    v_count    INT;
    v_user_ids UUID[];
BEGIN
    v_chat_id := COALESCE(NEW.chat_id, OLD.chat_id);

    SELECT type INTO v_type
    FROM chats
    WHERE id = v_chat_id;

    IF v_type IS DISTINCT FROM 'direct' THEN
        RETURN COALESCE(NEW, OLD);
    END IF;

    IF TG_OP = 'INSERT' THEN
        SELECT count(*) INTO v_count
        FROM chat_members
        WHERE chat_id = NEW.chat_id;

        IF v_count > 2 THEN
            RAISE EXCEPTION 'В direct-чате не может быть больше 2 участников';
        END IF;
    END IF;

    SELECT ARRAY(
        SELECT user_id
        FROM chat_members
        WHERE chat_id = v_chat_id
        ORDER BY user_id
    ) INTO v_user_ids;

    IF array_length(v_user_ids, 1) = 2 THEN
        UPDATE chats
        SET direct_key = v_user_ids[1]::text || '_' || v_user_ids[2]::text
        WHERE id = v_chat_id;
    ELSE
        UPDATE chats
        SET direct_key = NULL
        WHERE id = v_chat_id;
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_set_chat_direct_key
AFTER INSERT OR DELETE ON chat_members
FOR EACH ROW
EXECUTE FUNCTION set_chat_direct_key();

 
CREATE OR REPLACE FUNCTION prevent_remove_last_admin()
RETURNS TRIGGER AS $$
DECLARE
    v_admin_count INT;
BEGIN
    IF OLD.role = 'admin' THEN
        SELECT count(*) INTO v_admin_count
        FROM chat_members
        WHERE chat_id = OLD.chat_id AND role = 'admin' AND user_id != OLD.user_id;
 
        IF v_admin_count = 0 THEN
            RAISE EXCEPTION 'Нельзя удалить последнего администратора чата';
        END IF;
    END IF;
 
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
 
CREATE TRIGGER trg_prevent_remove_last_admin
BEFORE DELETE ON chat_members
FOR EACH ROW
EXECUTE FUNCTION prevent_remove_last_admin();

