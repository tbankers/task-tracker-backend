--was written by Artyom K., don't beat my meat

--enum types for role and status
CREATE TYPE member_role AS ENUM ('viewer', 'member', 'administrator');
CREATE TYPE task_status AS ENUM ('to_do', 'in_progress', 'done');
--users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP
);
--workspaces (where boards are made)
CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), 
    title TEXT, 
    created_by UUID REFERENCES users(id) ON DELETE SET NULL, 
    created_at TIMESTAMP
);
--ws members (which can view/write)
CREATE TABLE workspace_members (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE, 
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE, 
    role member_role
);
--boards (which contain tasks)
CREATE TABLE boards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), 
    name TEXT, 
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE, 
    created_at TIMESTAMP, 
    created_by UUID REFERENCES users(id) ON DELETE SET NULL
);
--tasks
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), 
    board_id UUID REFERENCES boards(id) ON DELETE CASCADE, 
    created_by UUID REFERENCES users(id) ON DELETE SET NULL, 
    created_at TIMESTAMP, 
    updated_at TIMESTAMP, 
    assigned_id UUID REFERENCES users(id) ON DELETE SET NULL, 
    name TEXT, 
    description TEXT, 
    status task_status DEFAULT 'to_do'
);
CREATE INDEX board_of_task ON tasks(board_id);
CREATE INDEX assigned_user ON tasks(assigned_id);
CREATE INDEX ws_member ON workspace_members(user_id);
CREATE UNIQUE INDEX user_email ON users(email);
