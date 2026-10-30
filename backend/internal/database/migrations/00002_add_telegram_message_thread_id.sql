-- +goose Up
ALTER TABLE settings ADD COLUMN telegram_message_thread_id INTEGER
    CHECK(telegram_message_thread_id IS NULL OR
        (typeof(telegram_message_thread_id) = 'integer' AND telegram_message_thread_id BETWEEN 1 AND 2147483647));
