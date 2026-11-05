package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Console struct {
	storage TaskStorage
	input   *bufio.Reader
	output  io.Writer
}

func NewConsole(storage TaskStorage, input io.Reader, output io.Writer) *Console {
	return &Console{storage: storage, input: bufio.NewReader(input), output: output}
}

func main() {
	console := NewConsole(&MemoryTaskStorage{}, os.Stdin, os.Stdout)
	if err := console.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка:", err)
		os.Exit(1)
	}
}

func (console *Console) Run() error {
	for {
		fmt.Fprint(console.output, "\n============ TASK MANAGER ============\n\n1. Создать задачу\n2. Показать все задачи\n3. Изменить статус\n4. Редактировать задачу\n5. Удалить задачу\n6. Найти задачу\n7. Статистика\n0. Выход\n\n")
		choice, err := console.readLine("Выберите действие: ")
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		switch choice {
		case "1":
			err = console.createTask()
		case "2":
			console.printTasks(console.storage.All())
		case "3":
			err = console.changeStatus()
		case "4":
			err = console.editTask()
		case "5":
			err = console.deleteTask()
		case "6":
			err = console.findTasks()
		case "7":
			console.printStatistics()
		case "0":
			fmt.Fprintln(console.output, "До свидания!")
			return nil
		default:
			err = errors.New("выберите пункт меню от 0 до 7")
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			fmt.Fprintln(console.output, "Ошибка:", err)
		}
	}
}

func (console *Console) readLine(prompt string) (string, error) {
	fmt.Fprint(console.output, prompt)
	line, err := console.input.ReadString('\n')
	if err != nil && !(errors.Is(err, io.EOF) && len(line) > 0) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (console *Console) readID() (int, error) {
	value, err := console.readLine("ID задачи: ")
	if err != nil {
		return 0, err
	}
	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		return 0, errors.New("ID должен быть положительным целым числом")
	}
	return id, nil
}

func (console *Console) readDetails() (Task, error) {
	title, err := console.readLine("Название: ")
	if err != nil {
		return Task{}, err
	}
	if title == "" {
		return Task{}, errors.New("название задачи не может быть пустым")
	}
	description, err := console.readLine("Описание: ")
	if err != nil {
		return Task{}, err
	}
	priority, err := console.readLine("Приоритет (Низкий, Средний, Высокий): ")
	if err != nil {
		return Task{}, err
	}
	return Task{Title: title, Description: description, Priority: priority}, nil
}

func (console *Console) createTask() error {
	details, err := console.readDetails()
	if err != nil {
		return err
	}
	task, err := console.storage.Create(details.Title, details.Description, details.Priority)
	if err != nil {
		return err
	}
	fmt.Fprintf(console.output, "Задача #%d успешно создана!\n", task.ID)
	return nil
}

func (console *Console) printTasks(tasks []Task) {
	if len(tasks) == 0 {
		fmt.Fprintln(console.output, "Задачи не найдены.")
		return
	}
	fmt.Fprintln(console.output, "ID | Название | Приоритет | Статус")
	for _, task := range tasks {
		fmt.Fprintf(console.output, "%d | %s | %s | %s\n", task.ID, task.Title, task.Priority, task.Status)
	}
}

func (console *Console) changeStatus() error {
	id, err := console.readID()
	if err != nil {
		return err
	}
	if _, err := console.storage.Get(id); err != nil {
		return err
	}
	status, err := console.readLine("Статус (Новая, В работе, Завершена): ")
	if err != nil {
		return err
	}
	if err := console.storage.SetStatus(id, status); err != nil {
		return err
	}
	fmt.Fprintf(console.output, "Статус задачи #%d изменён.\n", id)
	return nil
}

func (console *Console) editTask() error {
	id, err := console.readID()
	if err != nil {
		return err
	}
	if _, err := console.storage.Get(id); err != nil {
		return err
	}
	details, err := console.readDetails()
	if err != nil {
		return err
	}
	if err := console.storage.Update(id, details.Title, details.Description, details.Priority); err != nil {
		return err
	}
	fmt.Fprintf(console.output, "Задача #%d обновлена.\n", id)
	return nil
}

func (console *Console) deleteTask() error {
	id, err := console.readID()
	if err != nil {
		return err
	}
	if err := console.storage.Delete(id); err != nil {
		return err
	}
	fmt.Fprintf(console.output, "Задача #%d удалена.\n", id)
	return nil
}

func (console *Console) findTasks() error {
	fmt.Fprintln(console.output, "1. По названию\n2. По приоритету\n3. По статусу")
	choice, err := console.readLine("Выберите способ поиска: ")
	if err != nil {
		return err
	}
	var result []Task
	switch choice {
	case "1":
		query, readErr := console.readLine("Название или его часть: ")
		if readErr != nil {
			return readErr
		}
		result, err = SearchByTitle(console.storage.All(), query)
	case "2":
		priority, readErr := console.readLine("Приоритет (Низкий, Средний, Высокий): ")
		if readErr != nil {
			return readErr
		}
		result, err = FilterByPriority(console.storage.All(), priority)
	case "3":
		status, readErr := console.readLine("Статус (Новая, В работе, Завершена): ")
		if readErr != nil {
			return readErr
		}
		result, err = FilterByStatus(console.storage.All(), status)
	default:
		return errors.New("выберите способ поиска от 1 до 3")
	}
	if err != nil {
		return err
	}
	console.printTasks(result)
	return nil
}

func (console *Console) printStatistics() {
	statistics := CalculateStatistics(console.storage.All())
	fmt.Fprintf(console.output, "Всего задач: %d\nНовых: %d\nВ работе: %d\nЗавершённых: %d\nПроцент выполнения: %.2f%%\n", statistics.Total, statistics.New, statistics.Working, statistics.Done, statistics.CompletionPercent)
}
