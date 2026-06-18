-- CRUD functions query

-- name: CreateUser :one
INSERT INTO users (email, username, password_hash, email_verified, created_at) 
VALUES($1, $2, $3, FALSE, NOW()) 
RETURNING user_id;

-- name: GetUserById :one
SELECT user_id, email, username, password_hash, email_verified, created_at 
FROM users 
WHERE user_id = $1;

-- name: GetUserByEmail :one
SELECT user_id, email, username, password_hash, email_verified, created_at 
FROM users 
WHERE email = $1;

-- name: SetEmailVerified :exec
UPDATE users SET email_verified = TRUE WHERE user_id = $1;

-- name: ChangePassword :exec
UPDATE users SET password_hash = $1 WHERE user_id = $2;

-- name: GetWorkspaceById :one
SELECT workspace_id, title, created_at, created_by 
FROM workspaces 
WHERE workspace_id = $1;

-- name: CreateWorkspace :one
INSERT INTO workspaces (created_at, title, created_by) 
VALUES(NOW(), $1, $2) 
RETURNING workspace_id;

-- name: EditWorkspace :exec
UPDATE workspaces SET title = $1 WHERE workspace_id = $2;

-- name: DeleteWorkspace :exec
DELETE FROM workspaces WHERE workspace_id = $1;

-- name: GetUsersWorkspace :many
SELECT workspace_id, title, created_by, created_at 
FROM workspaces 
WHERE created_by = $1 
ORDER BY created_at ASC;

-- name: AddMember :one
INSERT INTO workspace_members (user_id, workspace_id, role) 
VALUES($1, $2, $3) 
RETURNING user_id;

-- name: ManageMember :exec
UPDATE workspace_members SET role = $1 WHERE user_id = $2;

-- name: KickUser :exec
DELETE FROM workspace_members WHERE user_id = $1;

-- name: GetMemberRoleById :one
SELECT role FROM workspace_members WHERE user_id = $1;

-- name: CreateBoard :one
INSERT INTO boards (title, workspace_id, created_at, created_by) 
VALUES($1, $2, NOW(), $3) 
RETURNING board_id;

-- name: GetWorkspaceBoards :many
SELECT board_id, title, workspace_id, created_at, created_by 
FROM boards 
WHERE workspace_id = $1 
ORDER BY created_at ASC;

-- name: EditBoard :exec
UPDATE boards SET title = $1 WHERE board_id = $2;

-- name: DeleteBoard :exec
DELETE FROM boards WHERE board_id = $1;

-- =========================================================================
-- NEW NEW NEW: COLUMNS CRUD (Your Dynamic Enum Management)
-- =========================================================================

-- name: CreateColumn :one
INSERT INTO columns (board_id, name, position, created_at)
VALUES ($1, $2, $3, NOW())
RETURNING column_id, board_id, name, position, created_at;

-- name: GetColumnsFromBoard :many
SELECT column_id, board_id, name, position, created_at
FROM columns
WHERE board_id = $1
ORDER BY position ASC, created_at ASC;

-- name: EditColumn :exec
UPDATE columns 
SET name = $1, position = $2 
WHERE column_id = $3;

-- name: DeleteColumn :exec
DELETE FROM columns WHERE column_id = $1;


-- =========================================================================
-- REFACTORED: TASKS CRUD (Uses column_id instead of board_id/status enum)
-- =========================================================================

-- name: CreateTask :one
INSERT INTO tasks (column_id, created_by, title, description, assigned_id) 
VALUES($1, $2, $3, $4, $5) 
RETURNING task_id;

-- name: MoveTaskToColumn :exec
UPDATE tasks SET column_id = $1, updated_at = NOW() WHERE task_id = $2;

-- name: UpdateTask :exec
UPDATE tasks 
SET title = $2, description = $3, assigned_id = $4, column_id = $5, updated_at = NOW() 
WHERE task_id = $1;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE task_id = $1;

-- name: GetTasksFromBoard :many
-- Uses a JOIN to fetch all tasks for a board now that board_id is gone from tasks table
SELECT t.task_id, t.column_id, t.created_at, t.created_by, t.updated_at, t.assigned_id, t.title, t.description
FROM tasks t
JOIN columns c ON t.column_id = c.column_id
WHERE c.board_id = $1 
ORDER BY t.created_at ASC;

-- name: GetTasksFromColumn :many
-- Keep this only if you intentionally want to paginate or lazy-load individual lists
SELECT task_id, column_id, created_at, created_by, updated_at, assigned_id, title, description 
FROM tasks 
WHERE column_id = $1 
ORDER BY created_at ASC;


-- =========================================================================
-- TOKENS & UTILITIES (Unchanged)
-- =========================================================================

-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (user_id, token, expires_at) 
VALUES ($1, $2, $3) 
RETURNING token_id;

-- name: GetPasswordResetToken :one
SELECT token_id, user_id, token, expires_at, created_at 
FROM password_reset_tokens 
WHERE token = $1;

-- name: DeletePasswordResetToken :exec
DELETE FROM password_reset_tokens WHERE token = $1;

-- name: DeleteExpiredTokens :exec
DELETE FROM password_reset_tokens WHERE expires_at < NOW();

-- name: GetTaskBlockpoints :many
SELECT blocked_by_task_id FROM task_blockpoints WHERE task_id = $1;

-- name: AddBlockpoint :exec
INSERT INTO task_blockpoints (task_id, blocked_by_task_id) 
VALUES ($1, $2) 
ON CONFLICT DO NOTHING;

-- name: RemoveBlockpoint :exec
DELETE FROM task_blockpoints WHERE task_id = $1 AND blocked_by_task_id = $2;

-- name: DeleteAllBlockpointsForTask :exec
DELETE FROM task_blockpoints WHERE task_id = $1;

-- name: CreateEmailVerificationToken :one
INSERT INTO email_verification_tokens (user_id, token, expires_at) 
VALUES ($1, $2, $3) 
RETURNING token_id;

-- name: GetEmailVerificationToken :one
SELECT token_id, user_id, token, expires_at, created_at 
FROM email_verification_tokens 
WHERE token = $1;

-- name: DeleteEmailVerificationToken :exec
DELETE FROM email_verification_tokens WHERE token = $1;

-- name: DeleteEmailVerificationTokensByUserID :exec
DELETE FROM email_verification_tokens WHERE user_id = $1;
