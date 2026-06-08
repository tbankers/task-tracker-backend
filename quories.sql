--CRUD functions query

-- name: getBoardTasks :many
SELECT id
FROM tasks 
WHERE board_id = $1
ORDER BY created_at ASC;

-- name: createTask :one
INSERT INTO tasks (created_at, name, description, assigned_id)
VALUES(NOW(), $1, $2, $3)
RETURNING id;

-- name: updateTask :exec
UPDATE tasks
SET updated_at = NOW(), name = $2, description = $3, assigned_id = $4, status = $5
WHERE id = $1;

-- name: deleteTask :exec
DELETE FROM tasks
WHERE id = $1;

-- name: getWorkspaceBoards :many
SELECT id
FROM boards
WHERE workspace_id = $1
ORDER BY created_at ASC;

-- name: createBoard :one
INSERT INTO boards (name, workspace_id, created_at, created_by)
VALUES($1, $2, NOW(), $3)
RETURNING id;