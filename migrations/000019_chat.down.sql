DROP INDEX IF EXISTS idx_chat_messages_order_created_at;
DROP INDEX IF EXISTS idx_chat_messages_thread_created_at;
DROP TABLE IF EXISTS chat_messages;
DROP INDEX IF EXISTS idx_chat_threads_order_id;
DROP INDEX IF EXISTS idx_chat_threads_passenger_support_unique;
DROP INDEX IF EXISTS idx_chat_threads_order_type_unique;
DROP TRIGGER IF EXISTS chat_threads_set_updated_at ON chat_threads;
DROP TABLE IF EXISTS chat_threads;
