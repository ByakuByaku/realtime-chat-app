CREATE OR REPLACE FUNCTION prevent_remove_last_admin()
RETURNS TRIGGER AS $$
DECLARE
    v_admin_count INT;
BEGIN
    IF OLD.role = 'admin' AND EXISTS (SELECT 1 FROM chats WHERE id = OLD.chat_id) THEN
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
