package main

import (
	"DB_Apps/pkg/model"
	"DB_Apps/pkg/myerrors"
	"DB_Apps/pkg/storage"
	"DB_Apps/pkg/storage/postgresql"
	"errors"
	"fmt"
	"log"
	"os"
)

var db storage.Interface

func main() {
	// Получаем пароль из переменной окружения
	pwd := os.Getenv("DB_pass")
	if pwd == "" {
		log.Fatal("Переменная окружения DB_pass не задана")
	}

	// Строка подключения
	connStr := fmt.Sprintf("postgres://postgres:%s@localhost:5432/tasks", pwd)

	var err error
	db, err = postgresql.New(connStr)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer db.Close()
	fillUsers()
	fillLabels()
	workWithTasks()

}

func fillUsers() {
	users := []model.User{
		{Name: "Иван Иванов"},
		{Name: "Мария Петрова"},
		{Name: "Алексей   сидОРов "}, // Проверка форматирования имени: Полсе форматирования в БД должно быть Алексей Сидоров
		{Name: ""},         // Ошибка: Пустая строка не проходит
		{Name: "John Doe"}, // Ошибка: В имени допускается только Кириллица
	}

	for _, u := range users {
		id, err := db.NewUser(u)
		if err != nil {
			switch {
			case errors.Is(err, postgresql.UserNameEmptyErr):
				fmt.Printf("Ошибка при добавлении пользователя %s: %v\n", u.Name, err)
				continue
			case errors.Is(err, postgresql.UserNameLangErr):
				fmt.Printf("Ошибка при добавлении пользователя %s: %v\n", u.Name, err)
				continue
			default:
				log.Fatalf("Ошибка при добавлении пользователя %s: %v\n", u.Name, err)
			}
		}

		fmt.Printf("Добавлен пользователь %s (ID %d)\n", u.Name, id)
	}
}

func fillLabels() {
	labels := []model.Label{
		{Name: "Ошибка"},
		{Name: "Новая"},
		{Name: "Срочно"},
		{Name: "Переделать"},
		{Name: "Идея"},
	}

	for _, l := range labels {
		id, err := db.NewLabel(l)
		if err != nil {
			if errors.Is(err, postgresql.LabelNameErr) {
				fmt.Printf("Ошибка при добавлении метки %s: %v\n", l.Name, err)
				continue
			}
			log.Fatalf("Ошибка при добавлении метки %s: %v\n", l.Name, err)
		}
		fmt.Printf("Добавлена метка %s (ID %d)\n", l.Name, id)
	}
}

func workWithTasks() {
	var t model.Task
	var err error
	var id int

	// Создание задачи без автора и меток
	t = model.Task{Title: "Ошибка при авторизации", Content: "При нажатии на кнопку \"Войти\" ничего не происходит"}
	id, err = db.NewTask(t)
	if err != nil {
		if e, ok := err.(myerrors.TaskPartialErr); ok {
			fmt.Println(e)
		} else {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("Создана задача с ID %d\n", id)
	}

	// Создание задачи без автора и одной меткой
	t = model.Task{
		Title:    "Баг в форме регистрации",
		Content:  "Кнопка \"Отправить\" не активна после заполнения всех полей",
		LabelsID: []int{5},
	}
	id, err = db.NewTask(t)
	if err != nil {
		if e, ok := err.(myerrors.TaskPartialErr); ok {
			fmt.Println(e)
		} else {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("Создана задача с ID %d\n", id)
	}

	// Создание задачи с автором и одной меткой
	t = model.Task{
		Title:      "Система уведомлений",
		Content:    "Реализовать уведомления при появлении новой новости",
		AuthorID:   1,
		AssignedID: 2,
		LabelsID:   []int{5},
	}
	id, err = db.NewTask(t)
	if err != nil {
		if e, ok := err.(myerrors.TaskPartialErr); ok {
			fmt.Println(e)
		} else {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("Создана задача с ID %d\n", id)
	}

	// Создание задачи с несколькими метками (часть неправильные)
	t = model.Task{
		Title:    "Добавить новый курс",
		Content:  "Добавить возможность пользователю приобрести новый курс",
		LabelsID: []int{2, 10, 3, 20},
	}
	id, err = db.NewTask(t)
	if err != nil {
		if e, ok := err.(myerrors.TaskPartialErr); ok {
			fmt.Println(e)
		} else {
			log.Fatal(err)
		}
	} else {
		fmt.Printf("Создана задача с ID %d\n", id)
	}

	// Получение всех задач
	fmt.Println("\nСписок всех задач:")
	tasks, err := db.SelectTasks()
	if err != nil {
		log.Fatal(err)
	}
	for _, task := range tasks {
		fmt.Printf("ID:%d | Title: %s | AuthorID: %v | AssignedID: %v | Content: %s\n",
			task.ID, task.Title, task.AuthorID, task.AssignedID, task.Content)
	}

	// Получение задач по автору
	fmt.Println("\nЗадачи по автору с ID 1:")
	tasks, err = db.SelectTasksByAuthorID(1)
	if err != nil {
		log.Fatal(err)
	}
	for _, task := range tasks {
		fmt.Printf("ID:%d | Title: %s | Content: %s\n", task.ID, task.Title, task.Content)
	}

	// Получение задач по метке
	fmt.Println("\nЗадачи с меткой ID 5:")
	tasks, err = db.SelectTasksByLabelID(5)
	if err != nil {
		log.Fatal(err)
	}
	for _, task := range tasks {
		fmt.Printf("ID:%d | Title: %s\n", task.ID, task.Title)
	}

	// Обновление задачи
	fmt.Println("\nОбновление задачи ID 1...")
	updateTask := model.Task{
		ID:         1,
		AuthorID:   1, // автор менять нельзя
		AssignedID: 3,
		Title:      "Ошибка при авторизации",
		Content:    "Проверить обработчик кнопки и запрос",
	}
	if err := db.UpdateTaskByID(updateTask); err != nil {
		fmt.Println("Ошибка при обновлении задачи:", err)
	} else {
		fmt.Println("Задача успешно обновлена")
	}

	// Получение всех задач
	fmt.Println("\nСписок всех задач:")
	tasks, err = db.SelectTasks()
	if err != nil {
		log.Fatal(err)
	}
	for _, task := range tasks {
		fmt.Printf("ID:%d | Title: %s | AuthorID: %v | AssignedID: %v | Content: %s\n",
			task.ID, task.Title, task.AuthorID, task.AssignedID, task.Content)
	}

	// Удаление задачи
	fmt.Println("\nУдаление задачи ID 2...")
	if err := db.DeleteTask(2); err != nil {
		fmt.Println("Ошибка при удалении:", err)
	} else {
		fmt.Println("Задача успешно удалена")
	}

	// Проверка всех задач после удаления
	fmt.Println("\nСписок задач после удаления:")
	tasks, err = db.SelectTasks()
	if err != nil {
		log.Fatal(err)
	}
	for _, task := range tasks {
		fmt.Printf("ID:%d | Title: %s\n", task.ID, task.Title)
	}
}
