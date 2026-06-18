-- =========================================================================
-- 1. CREATE THE NEW COLUMNS TABLE (The New Dynamic Status Enum)
-- =========================================================================
CREATE TABLE columns (
    column_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id UUID NOT NULL REFERENCES boards(board_id) ON DELETE CASCADE,
    name TEXT NOT NULL, -- This acts as your dynamic 'task_status' (e.g., 'To Do', 'In Review')
    position INT NOT NULL DEFAULT 0, -- Highly recommended for sorting columns left-to-right
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for fast loading of a board's column layout
CREATE INDEX idx_columns_board_id ON columns(board_id);

-- =========================================================================
-- 2. REFACTOR THE TASKS TABLE
-- =========================================================================

-- Step A: Remove the old static status column
ALTER TABLE tasks DROP COLUMN status;

-- Step B: Remove the old board_id column (since tasks now belong to a specific column)
-- Also clean up its old index
DROP INDEX board_of_task;
ALTER TABLE tasks DROP COLUMN board_id;

-- Step C: Add the new column_id reference to tasks
-- This acts as your foreign key validation to the dynamic statuses
ALTER TABLE tasks ADD COLUMN column_id UUID REFERENCES columns(column_id) ON DELETE CASCADE;

-- Step D: Create a fast lookup index for fetching tasks inside a column
CREATE INDEX idx_tasks_column_id ON tasks(column_id);

-- =========================================================================
-- 3. CLEAN UP ARTYOM'S SCHEMA TYPE (Optional but tidy)
-- =========================================================================
-- We no longer need the hardcoded task_status enum type anywhere in the system.
DROP TYPE task_status;
