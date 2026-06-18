-- 1. Создаем функцию, которая будет вставлять 3 колонки
CREATE OR REPLACE FUNCTION create_default_columns_for_board()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO columns (board_id, name, position)
    VALUES 
        (NEW.board_id, 'To Do', 0),
        (NEW.board_id, 'In Progress', 1),
        (NEW.board_id, 'Done', 2);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 2. Вешаем эту функцию как триггер на таблицу boards
CREATE TRIGGER after_board_created
AFTER INSERT ON boards
FOR EACH ROW
EXECUTE FUNCTION create_default_columns_for_board();
