--was written by Artyom K., don't beat my meat 

--enum types for role and status 
CREATE TYPE member_role AS ENUM ('viewer', 'member', 'administrator'); 
CREATE TYPE task_status AS ENUM ('to_do', 'in_progress', 'done'); 

--users 
CREATE TABLE users ( 
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(), 
    email TEXT NOT NULL UNIQUE, 
    username TEXT NOT NULL, 
    password_hash TEXT NOT NULL, 
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
); 

--workspaces (where boards are made) 
CREATE TABLE workspaces ( 
    workspace_id UUID PRIMARY KEY DEFAULT gen_random_uuid(), 
    title TEXT, 
    created_by UUID REFERENCES users(user_id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
); 

--ws members (which can view/write) 
CREATE TABLE workspace_members ( 
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    workspace_id UUID REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
    role member_role,
    PRIMARY KEY (user_id, workspace_id)
); 

--boards (which contain tasks) 
CREATE TABLE boards ( 
    board_id UUID PRIMARY KEY DEFAULT gen_random_uuid(), 
    title TEXT, 
    workspace_id UUID REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
    created_by UUID REFERENCES users(user_id) ON DELETE SET NULL
); 

--tasks 
CREATE TABLE tasks ( 
    task_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, 
    board_id UUID REFERENCES boards(board_id) ON DELETE CASCADE,
    created_by UUID REFERENCES users(user_id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
    assigned_id UUID REFERENCES users(user_id) ON DELETE SET NULL,
    title TEXT, 
    description TEXT, 
    status task_status DEFAULT 'to_do' 
); 

CREATE INDEX board_of_task ON tasks(board_id); 
CREATE INDEX assigned_user ON tasks(assigned_id); 
CREATE INDEX ws_member ON workspace_members(user_id); 
