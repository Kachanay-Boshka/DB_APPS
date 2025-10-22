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
	return fmt.Sprintf("\n\t- Ошибка: %s", strings.Join(msgs, "\n\t- Ошибка: "))
}
