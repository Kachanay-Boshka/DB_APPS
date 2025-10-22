package postgresql

import (
	"DB_Apps/pkg/model"
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v4"
)

// UserNameErr возвращается, если имя пользователя имеет неправильный формат
var UserNameLangErr = errors.New("В имени допускается только Кириллица")
var UserNameEmptyErr = errors.New("Пустое поле имени")

// NewUser создает нового пользователя в таблице users
// Проверяет корректность имени, форматирует его и возвращает ID созданного пользователя
func (s *Storage) NewUser(user model.User) (int, error) {
	var id int
	if err := checkName(user.Name); err != nil {
		return 0, err
	}
	nameConversion(&user.Name)

	err := s.db.QueryRow(context.Background(), "INSERT INTO users(name) VALUES ($1) RETURNING id;", user.Name).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// DeleteUser удаляет пользователя по ID
// Если пользователь не найден, то возвращает ошибку
func (s *Storage) DeleteUser(id int) error {
	r, err := s.db.Exec(context.Background(), "DELETE FROM users WHERE id = $1;", id)
	if err != nil {
		return err
	}
	if r.RowsAffected() == 0 {
		return fmt.Errorf("Пользователя с ID: %d не найден", id)
	}
	return nil
}

// SelectUsers возвращает список всех пользователей, отсортированных по ID
// Если пользователей нет, то возвращает пустой срез и ошибку
func (s *Storage) SelectUsers() ([]model.User, error) {
	var users []model.User
	rows, err := s.db.Query(context.Background(), "SELECT id, name FROM users ORDER BY id ASC;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user model.User
		err := rows.Scan(&user.ID, &user.Name)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil

}

// SelectUserByID возвращает пользователя по ID
// Если пользователь не найден, то возвращает ошибку
func (s *Storage) SelectUserByID(id int) (model.User, error) {
	var user model.User
	err := s.db.QueryRow(context.Background(), "SELECT id, name FROM users WHERE id = $1;", id).Scan(&user.ID, &user.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user, fmt.Errorf("Пользователь с ID %d не найден", id)
		}
		return user, err
	}
	return user, nil
}

// UpdateUserName изменяет имя пользователя по ID
// Проверяет корректность имени и форматирует его
// Если пользователь не найден, то возвращает ошибку
func (s *Storage) UpdateUserName(id int, newName string) error {
	if err := checkName(newName); err != nil {
		return err
	}
	nameConversion(&newName)
	r, err := s.db.Exec(context.Background(), "UPDATE users SET name = $1 WHERE id = $2;", newName, id)
	if err != nil {
		return err
	}
	if r.RowsAffected() == 0 {
		return fmt.Errorf("Пользователь с ID %d не найден", id)
	}
	return nil
}

// checkName проверяет корректность имени пользователя:
// - Имя должно состоять только из кириллических символов и пробелов
// - Возвращает false, если имя пустое или содержит недопустимые символы
func checkName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return UserNameEmptyErr
	}

	for _, w := range name {
		if !(w >= 'а' && w <= 'я' || w >= 'А' && w <= 'Я' || w == ' ') {
			return UserNameLangErr
		}
	}

	return nil
}

// nameConversion форматирует имя пользователя:
// - Приводит его к виду, что каждое слово с заглавной буквы
func nameConversion(name *string) {
	*name = strings.ToLower(*name)
	if len(*name) > 0 {
		words := strings.Fields(*name)
		for i := range words {
			runnes := []rune(words[i])
			runnes[0] = unicode.ToUpper(runnes[0])
			words[i] = string(runnes)
		}
		*name = strings.Join(words, " ")
	}
}
