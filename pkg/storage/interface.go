package storage

import (
	"DB_Apps/pkg/model"
)

type Interface interface {
	// Для работы с пользователем(users)
	NewUser(model.User) (int, error)
	DeleteUser(int) error
	UpdateUserName(int, string) error
	SelectUsers() ([]model.User, error)
	SelectUserByID(int) (model.User, error)

	// Для работы с метками(labels)
	NewLabel(model.Label) (int, error)
	DeleteLabel(int) error
	UpdateLabelName(int, string) error
	SelectLabels() ([]model.Label, error)
	SelectLabelByID(int) (model.Label, error)

	// Для работы с задачами(tasks)
	NewTask(model.Task) (int, error)
	SelectTasks() ([]model.Task, error)
	SelectTasksByAuthorID(int) ([]model.Task, error)
	SelectTasksByLabelID(int) ([]model.Task, error)
	DeleteTask(int) error
	UpdateTaskByID(model.Task) error
	AddLabelToTask(int, int) error
	DeleteLabelToTask(int, int) error

	// Закрытие соедининя с БД
	Close()
}
