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
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
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
    status task_status DEFAULT 'to_do',
    start_date DATE,
    due_date DATE
); 

--password reset tokens
CREATE TABLE password_reset_tokens (
    token_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_password_reset_tokens_token ON password_reset_tokens(token);
CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);

--email verification tokens
CREATE TABLE email_verification_tokens (
    token_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_email_verification_tokens_token ON email_verification_tokens(token);
CREATE INDEX idx_email_verification_tokens_user_id ON email_verification_tokens(user_id);

--task_blockpoints (task dependencies: blocked_task cannot be done until blocker_task is done)
CREATE TABLE task_blockpoints (
    task_id INT NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    blocked_by_task_id INT NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, blocked_by_task_id)
);

CREATE INDEX board_of_task ON tasks(board_id); 
CREATE INDEX assigned_user ON tasks(assigned_id); 
CREATE INDEX ws_member ON workspace_members(user_id); 
