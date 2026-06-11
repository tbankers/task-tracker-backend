--CRUD functions query

-- name: GetBoardTasks :many
SELECT id
FROM tasks 
WHERE board_id = $1
ORDER BY created_at ASC;

-- name: CreateTask :one
INSERT INTO tasks (created_at, name, description, assigned_id)
VALUES(NOW(), $1, $2, $3)
RETURNING id;

-- name: UpdateTask :exec
UPDATE tasks
SET updated_at = NOW(), name = $2, description = $3, assigned_id = $4, status = $5
WHERE id = $1;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE id = $1;

-- name: GetWorkspaceBoards :many
SELECT id
FROM boards
WHERE workspace_id = $1
ORDER BY created_at ASC;

-- name: CreateBoard :one
INSERT INTO boards (name, workspace_id, created_at, created_by)
VALUES($1, $2, NOW(), $3)
RETURNING id;