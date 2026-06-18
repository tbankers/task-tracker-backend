-- comments
CREATE TABLE
    comments (
        comment_id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
        board_id UUID REFERENCES boards (board_id) ON DELETE CASCADE,
        author_id UUID REFERENCES users (user_id) ON DELETE SET NULL,
        sent_at TIMESTAMP DEFAULT NOW(),
        content TEXT
    );

CREATE INDEX idx_comments_board_id ON comments (board_id);