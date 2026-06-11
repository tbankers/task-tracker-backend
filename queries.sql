--CRUD functions query


-- name: CreateUser :one
INSERT INTO users (email, username, password_hash, created_at)
VALUES($1, $2, $3, NOW())
RETURNING user_id;

-- name: GetUserById :one
SELECT user_id, email, username, password_hash, created_at
FROM users
WHERE user_id = $1;

-- name: GetUserByEmail :one
SELECT user_id, email, username, password_hash, created_at
FROM users
WHERE email = $1;

-- name: ChangePassword :exec
UPDATE users 
SET password_hash = $1
WHERE user_id = $2;

-- name: GetWorkspaceById :one
SELECT workspace_id, title, created_at, created_by
FROM workspaces
WHERE workspace_id = $1;

-- name: CreateWorkspace :one
INSERT INTO workspaces (created_at, title, created_by)
VALUES(NOW(), $1, $2)
RETURNING workspace_id;

-- name: EditWorkspace :exec
UPDATE workspaces 
SET title = $1
WHERE workspace_id = $2;

-- name: DeleteWorkspace :one
DELETE FROM workspaces
WHERE workspace_id = $1;

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
UPDATE workspace_members
SET role = $1
WHERE user_id = $2;

-- name: KickUser :exec
DELETE FROM workspace_members
WHERE user_id = $1;

-- name: GetMemberRoleById :one
SELECT role
FROM workspace_members
WHERE user_id = $1;

-- name: CreateBoard :one
INSERT INTO boards (title, workspace_id, created_at, created_by)
VALUES($1, $2, NOW(), $3)
RETURNING board_id;

-- name: GetWorkspaceBoards :many
SELECT board_id
FROM boards
WHERE workspace_id = $1
ORDER BY created_at ASC;

-- name: EditBoard :exec
UPDATE boards
SET title = $1
WHERE board_id = $2;

-- name: DeleteBoard :exec
DELETE FROM boards
WHERE board_id = $1;

-- name: CreateTask :one
INSERT INTO tasks (board_id, created_by, title, description, assigned_id)
VALUES($1, $2, $3, $4, $5)
RETURNING task_id;

-- name: ChangeTaskStatus :exec
UPDATE tasks
SET status = $1
WHERE task_id = $2;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE task_id = $1;

-- name: GetTasksFromBoard :many
SELECT task_id, board_id, created_at, created_by, updated_at, assigned_id, title, description, status
FROM tasks 
WHERE board_id = $1
ORDER BY created_at ASC;