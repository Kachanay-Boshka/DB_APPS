package postgresql

import (
	"DB_Apps/pkg/model"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
)

// Ошибка: метка не может быть пустой
var LabelNameErr = errors.New("Пустая строка не может быть меткой")

// NewLabek создает новую метку в таблице Labels
// Возвращает ID созданной метки
// Если имя метки пустое значение, то возвращает ошибку LabelNameErr
func (s *Storage) NewLabel(l model.Label) (int, error) {
	var id int
	err := checkLabelName(&l.Name)
	if err != nil {
		return 0, err
	}
	err = s.db.QueryRow(context.Background(), "INSERT INTO labels(name) VALUES ($1) RETURNING id;", l.Name).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// DeleteLabel удаляет метку по ID
// Если метка не найдена, то возвращает ошибку
func (s *Storage) DeleteLabel(id int) error {
	r, err := s.db.Exec(context.Background(), "DELETE FROM labels WHERE id = $1;", id)
	if err != nil {
		return err
	}

	if r.RowsAffected() == 0 {
		return fmt.Errorf("Метка с ID: %d не найдена", id)
	}

	return nil
}

// UpdateLabelName обновляет имя метки по ее ID
// Проверяет новое имя на корректность
// Если метка не найдена, то возвращает ошибку
func (s *Storage) UpdateLabelName(id int, newName string) error {
	err := checkLabelName(&newName)
	if err != nil {
		return err
	}
	r, err := s.db.Exec(context.Background(), "UPDATE labels SET name = $1 WHERE id = $2;", newName, id)
	if err != nil {
		return err
	}

	if r.RowsAffected() == 0 {
		return fmt.Errorf("Метка с ID %d не найдена", id)
	}
	return nil
}

// SelectLabels возвращает список всех меток в порядке возрастания ID
// Если меток нет, то возвращает пустой срез
func (s *Storage) SelectLabels() ([]model.Label, error) {
	var labels []model.Label
	rows, err := s.db.Query(context.Background(), "SELECT id, name FROM labels ORDER BY id ASC;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var label model.Label
		err := rows.Scan(&label.ID, &label.Name)
		if err != nil {
			return nil, err
		}
		labels = append(labels, label)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return labels, nil
}

// SelectLabelByID возвращает метку по ее ID
// Если метка не найдена, то возвращает ошибку
func (s *Storage) SelectLabelByID(id int) (model.Label, error) {
	var label model.Label
	err := s.db.QueryRow(context.Background(), "SELECT id, name FROM labels WHERE id = $1", id).Scan(&label.ID, &label.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return label, fmt.Errorf("Пользователь с ID %d не найден", id)
		}
		return label, err
	}

	return label, nil
}

// checkLabelName проверяет корректность имени метки:
// - Удаляет лишние пробелы спереди и сзади
// - Сводит подряд идущие пробелы к одному
// - Возвращает ошибку, если строка пустая
func checkLabelName(name *string) error {
	*name = strings.TrimSpace(*name)
	if len(*name) < 1 {
		return LabelNameErr
	}
	*name = strings.Join(strings.Fields(*name), " ")

	return nil
}
