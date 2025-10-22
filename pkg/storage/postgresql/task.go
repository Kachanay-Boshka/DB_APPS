package postgresql

import (
	"DB_Apps/pkg/model"
	"DB_Apps/pkg/myerrors"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

const (
	ForeignKeyViolation = "23503"
	UniqueViolation     = "23505"
)

// Ошибка при добавление к существующей задаче
var LabelOrTaskNotExistErr = errors.New("Задачи или метки не существует")

// Ошибка при добавлении дубликата метки к задаче
var DuplicateLabelIDErr = errors.New("Метка уже существует")

// NewTask создает новую задачу и возвращает е ID
// Перед вставкой очищает поля title и content от лишних пробелов
func (s *Storage) NewTask(task model.Task) (int, error) {
	var id int
	task.Title = strings.TrimSpace(task.Title)
	task.Content = strings.TrimSpace(task.Content)

	tx, err := s.db.Begin(context.Background())
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(context.Background())

	var errs myerrors.TaskPartialErr

	// Проверка сущестовавания меток
	for _, labelID := range task.LabelsID {
		var exists bool
		err := tx.QueryRow(context.Background(),
			`SELECT EXISTS(SELECT 1 FROM labels WHERE id = $1)`, labelID).Scan(&exists)
		if err != nil {
			return 0, fmt.Errorf("Ошибка при проверке метки %d: %w", labelID, err)
		}
		if !exists {
			errs.Errs = append(errs.Errs, fmt.Errorf("Метка с ID:%d не существует", labelID))
		}
	}

	if len(errs.Errs) > 0 {
		return 0, fmt.Errorf("Ошибка создания задачи: %w", errs)
	}

	err = tx.QueryRow(context.Background(),
		`INSERT INTO tasks(author_id, assigned_id, title, content) VALUES ($1, $2, $3, $4) RETURNING id;`,
		task.AuthorID, task.AssignedID, task.Title, task.Content).Scan(&id)

	if err != nil {
		if e, ok := err.(*pgconn.PgError); ok && e.Code == ForeignKeyViolation {
			return 0, fmt.Errorf("автора(AuthorID) с ID:%d или ответственного(AssignedID) с ID:%d не существует",
				task.AuthorID, task.AssignedID)
		}
		return 0, err
	}

	// Добавление меток (если они ВСЕ существуют)
	for _, labelID := range task.LabelsID {
		if err := s.syncLabelToTask(tx, labelID, id); err != nil {
			return 0, fmt.Errorf("Неожиданная ошибка при добавлении метки %d: %w", labelID, err)
		}
	}

	if err = tx.Commit(context.Background()); err != nil {
		return 0, fmt.Errorf("ошибка при коммите транзакции: %w", err)
	}

	return id, nil
}

// SelectTasks возвращает список всех задач, отсортированных по ID
func (s *Storage) SelectTasks() ([]model.Task, error) {
	var tasks []model.Task
	rows, err := s.db.Query(context.Background(), `SELECT id, opened, closed, author_id, assigned_id,
														title, content FROM tasks ORDER BY id ASC;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var task model.Task
		err = rows.Scan(
			&task.ID,
			&task.Opened,
			&task.Closed,
			&task.AuthorID,
			&task.AssignedID,
			&task.Title,
			&task.Content,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

// SelectTasksByAuthorID возвращает все задачи, созданные конкретным автором
func (s *Storage) SelectTasksByAuthorID(authorID int) ([]model.Task, error) {
	rows, err := s.db.Query(context.Background(), `SELECT id, opened, closed, author_id, assigned_id,
														title, content FROM tasks WHERE author_id = $1 ORDER BY id ASC;`, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []model.Task
	for rows.Next() {
		var task model.Task
		err = rows.Scan(
			&task.ID,
			&task.Opened,
			&task.Closed,
			&task.AuthorID,
			&task.AssignedID,
			&task.Title,
			&task.Content,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

// SelectTasksByLabelID возвращает все задачи, связанные с конкретной меткой
func (s *Storage) SelectTasksByLabelID(labelID int) ([]model.Task, error) {
	rows, err := s.db.Query(context.Background(), `SELECT tasks.id, tasks.opened, 
														tasks.closed, tasks.author_id, 
														tasks.assigned_id, tasks.title, tasks.content
														FROM tasks JOIN tasks_labels 
														ON tasks.id = tasks_labels.task_id 
														WHERE tasks_labels.label_id = $1;`, labelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []model.Task
	for rows.Next() {
		var task model.Task
		err = rows.Scan(
			&task.ID,
			&task.Opened,
			&task.Closed,
			&task.AuthorID,
			&task.AssignedID,
			&task.Title,
			&task.Content,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

// DeleteTask удаляет задачу по ID
// Возвращает ошибку, если задача не найдена
func (s *Storage) DeleteTask(id int) error {
	tx, err := s.db.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	_, err = tx.Exec(context.Background(), "DELETE FROM tasks_labels WHERE task_id = $1;", id)
	if err != nil {
		return fmt.Errorf("Ошибка при удалении связей задачи %d: %w", id, err)
	}

	r, err := tx.Exec(context.Background(), "DELETE FROM tasks WHERE id = $1;", id)
	if err != nil {
		return fmt.Errorf("Ошибка при удалении задачи %d: %w", id, err)
	}

	if r.RowsAffected() == 0 {
		return fmt.Errorf("Задача с ID %d не найдена", id)
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("Ошибка при созранении результатов транзакции: %w", err)
	}

	return nil
}

// UpdateTaskByID обновляет поля задачи (автора, исполнителя, заголовок, описание)
// Перед обновлением очищает текстовые поля от пробелов
// Возвращает ошибку, если задача с указанным ID не найдена
func (s *Storage) UpdateTaskByID(task model.Task) error {
	tx, err := s.db.Begin(context.Background())
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())

	var currentAuthorID int
	err = tx.QueryRow(context.Background(),
		`SELECT author_id FROM tasks WHERE id = $1;`, task.ID).Scan(&currentAuthorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("Задача с ID %d не найдена", task.ID)
		}
		return fmt.Errorf("Ошибка при получении задачи: %w", task.ID, err)
	}

	if currentAuthorID != 0 && currentAuthorID != task.AuthorID {
		return fmt.Errorf("Нельзя менять автора: текущий автор задачи имеет ID %d", currentAuthorID)
	}

	var assignedExists bool
	err = tx.QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1);`, task.AssignedID).Scan(&assignedExists)
	if err != nil {
		return fmt.Errorf("Ошибка при проверке исполнителя: %w", err)
	}
	if !assignedExists {
		return fmt.Errorf("Исполнитель с ID %d не существует", task.AssignedID)
	}

	var errs myerrors.TaskPartialErr
	for _, labelID := range task.LabelsID {
		var labelExists bool
		err := tx.QueryRow(context.Background(),
			`SELECT EXISTS(SELECT 1 FROM labels WHERE id = $1)`, labelID).Scan(&labelExists)
		if err != nil {
			return fmt.Errorf("Ошибка при проверке метки %d: %w", labelID, err)
		}
		if !labelExists {
			errs.Errs = append(errs.Errs, fmt.Errorf("Метка с ID:%d не существует", labelID))
		}
	}

	if len(errs.Errs) > 0 {
		return errs
	}

	task.Title = strings.TrimSpace(task.Title)
	task.Content = strings.TrimSpace(task.Content)

	r, err := tx.Exec(context.Background(), `UPDATE tasks
		SET assigned_id = $1,
			title = $2,
			content = $3
		WHERE id = $4;`,
		task.AssignedID, task.Title, task.Content, task.ID)
	if err != nil {
		return fmt.Errorf("Ошибка при обновлении задачи: %w", err)
	}

	if r.RowsAffected() == 0 {
		return fmt.Errorf("Задача с ID %d не найдена", task.ID)
	}

	// Удаление старых меток
	_, err = tx.Exec(context.Background(),
		`DELETE FROM tasks_labels WHERE task_id = $1;`, task.ID)
	if err != nil {
		return fmt.Errorf("Ошибка при удалении старых меток: %w", err)
	}

	// Добавление новых меток
	for _, labelID := range task.LabelsID {
		if err := s.syncLabelToTask(tx, labelID, task.ID); err != nil {
			return fmt.Errorf("Неожиданная ошибка при добавлении метки %d: %w", labelID, err)
		}
	}

	if err := tx.Commit(context.Background()); err != nil {
		return fmt.Errorf("Ошибка при сохранении результатов транзакции: %w", err)
	}

	return nil
}

// AddLabelToTask добавляет метку к задаче
// Если такая связь уже существует, то возвращает ошибку
func (s *Storage) AddLabelToTask(id_label, id_task int) error {
	_, err := s.db.Exec(context.Background(), `INSERT INTO tasks_labels (task_id, label_id) 
												VALUES ($1, $2);`, id_task, id_label)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			switch pgErr.Code {
			case UniqueViolation:
				// Нарушение UNIQUE
				return fmt.Errorf("Предупреждение для метки с ID:%d для задачи с ID:%d: %w", id_label, id_label, DuplicateLabelIDErr)
			case ForeignKeyViolation:
				// Нарушение внешнего ключа
				return fmt.Errorf("Ошибка связи для задачи с ID:%d и метки с ID:%d: %w ", id_task, id_label, LabelOrTaskNotExistErr)
			}
		}
		return err
	}
	return nil
}

// syncLabelToTask добавляет метку к задаче используя транзакцию
// Если такая связь уже существует, то возвращает ошибку
func (s *Storage) syncLabelToTask(tx pgx.Tx, id_label, id_task int) error {
	_, err := tx.Exec(context.Background(),
		`INSERT INTO tasks_labels (task_id, label_id) VALUES ($1, $2);`,
		id_task, id_label)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			switch pgErr.Code {
			case UniqueViolation:
				return fmt.Errorf("Предупреждение для метки с ID:%d для задачи с ID:%d: %w", id_label, id_task, DuplicateLabelIDErr)
			case ForeignKeyViolation:
				return fmt.Errorf("%w: задача с ID:%d или метка с ID:%d не существуют",
					LabelOrTaskNotExistErr, id_task, id_label)
			}
		}
		return err
	}
	return nil
}

// DeleteLabelToTask удаляет связь между задачей и меткой
// Если связь отсутствует, то возвращает ошибку
func (s *Storage) DeleteLabelToTask(id_label, id_task int) error {
	r, err := s.db.Exec(context.Background(), `DELETE FROM tasks_labels  
												WHERE task_id = $1 AND label_id = $2`,
		id_task, id_label)
	if err != nil {
		return err
	}
	if r.RowsAffected() == 0 {
		return fmt.Errorf("Метка с ID:%d не существовало для задачи с ID:%d", id_label, id_task)
	}
	return nil
}
