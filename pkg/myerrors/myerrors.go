package myerrors

import (
	"fmt"
	"strings"
)

type TaskPartialErr struct {
	TaskID int
	Errs   []error
}

func (e TaskPartialErr) Error() string {
	msgs := make([]string, len(e.Errs))
	for i, err := range e.Errs {
		msgs[i] = err.Error()
	}
	return fmt.Sprintf("Задача c ID:%d создана, но есть ошибки:\n\t- Ошибка:%s", e.TaskID, strings.Join(msgs, "\n\t- Ошибка:"))
}
