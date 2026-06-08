CREATE INDEX board_of_task ON tasks(board_id);
CREATE INDEX assigned_user ON tasks(assigned_id);
CREATE INDEX ws_member ON workspace_members(user_id);
CREATE UNIQUE INDEX user_email ON users(email);