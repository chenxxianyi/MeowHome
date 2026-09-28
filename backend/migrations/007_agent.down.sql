-- +migrate Down
DROP TABLE IF EXISTS ai_agent_messages;
DROP TABLE IF EXISTS ai_agent_sessions;
-- 保留 reminders 的兼容字段，避免回滚 Agent 时破坏已有提醒数据。
