package main

import (
	"errors"
	"fmt"
	"strings"
)

const (
	PriorityLow    = "Низкий"
	PriorityMedium = "Средний"
	PriorityHigh   = "Высокий"
	StatusNew      = "Новая"
	StatusWorking  = "В работе"
	StatusDone     = "Завершена"
)

type Task struct {
	ID          int
	Title       string
	Description string
	Priority    string
	Status      string
}

type TaskStorage interface {
	Create(title, description, priority string) (Task, error)
	All() []Task
	Get(id int) (Task, error)
	Update(id int, title, description, priority string) error
	SetStatus(id int, status string) error
	Delete(id int) error
}

type MemoryTaskStorage struct {
	tasks  []Task
	lastID int
}

type Statistics struct {
	Total             int
	New               int
	Working           int
	Done              int
	CompletionPercent float64
}

func normalizeChoice(value string, choices ...string) (string, error) {
	value = strings.TrimSpace(value)
	for _, choice := range choices {
		if strings.EqualFold(value, choice) {
			return choice, nil
		}
	}
	return "", fmt.Errorf("допустимые значения: %s", strings.Join(choices, ", "))
}

func (task *Task) validateDetails() error {
	task.Title = strings.TrimSpace(task.Title)
	if task.Title == "" {
		return errors.New("название задачи не может быть пустым")
	}
	priority, err := normalizeChoice(task.Priority, PriorityLow, PriorityMedium, PriorityHigh)
	if err != nil {
		return fmt.Errorf("недопустимый приоритет: %w", err)
	}
	task.Description = strings.TrimSpace(task.Description)
	task.Priority = priority
	return nil
}

func (storage *MemoryTaskStorage) Create(title, description, priority string) (Task, error) {
	task := Task{Title: title, Description: description, Priority: priority, Status: StatusNew}
	if err := task.validateDetails(); err != nil {
		return Task{}, err
	}
	storage.lastID++
	task.ID = storage.lastID
	storage.tasks = append(storage.tasks, task)
	return task, nil
}

func (storage *MemoryTaskStorage) All() []Task {
	return append([]Task(nil), storage.tasks...)
}

func (storage *MemoryTaskStorage) findIndex(id int) (int, error) {
	for index, task := range storage.tasks {
		if task.ID == id {
			return index, nil
		}
	}
	return -1, fmt.Errorf("задача с ID %d не найдена", id)
}

func (storage *MemoryTaskStorage) Get(id int) (Task, error) {
	index, err := storage.findIndex(id)
	if err != nil {
		return Task{}, err
	}
	return storage.tasks[index], nil
}

func (storage *MemoryTaskStorage) Update(id int, title, description, priority string) error {
	index, err := storage.findIndex(id)
	if err != nil {
		return err
	}
	task := storage.tasks[index]
	task.Title = title
	task.Description = description
	task.Priority = priority
	if err := task.validateDetails(); err != nil {
		return err
	}
	storage.tasks[index] = task
	return nil
}

func (storage *MemoryTaskStorage) SetStatus(id int, status string) error {
	index, err := storage.findIndex(id)
	if err != nil {
		return err
	}
	status, err = normalizeChoice(status, StatusNew, StatusWorking, StatusDone)
	if err != nil {
		return fmt.Errorf("недопустимый статус: %w", err)
	}
	storage.tasks[index].Status = status
	return nil
}

func (storage *MemoryTaskStorage) Delete(id int) error {
	index, err := storage.findIndex(id)
	if err != nil {
		return err
	}
	copy(storage.tasks[index:], storage.tasks[index+1:])
	storage.tasks[len(storage.tasks)-1] = Task{}
	storage.tasks = storage.tasks[:len(storage.tasks)-1]
	return nil
}

func SearchByTitle(tasks []Task, query string) ([]Task, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, errors.New("поисковый запрос не может быть пустым")
	}
	var result []Task
	for _, task := range tasks {
		if strings.Contains(strings.ToLower(task.Title), query) {
			result = append(result, task)
		}
	}
	return result, nil
}

func FilterByPriority(tasks []Task, priority string) ([]Task, error) {
	priority, err := normalizeChoice(priority, PriorityLow, PriorityMedium, PriorityHigh)
	if err != nil {
		return nil, fmt.Errorf("недопустимый приоритет: %w", err)
	}
	var result []Task
	for _, task := range tasks {
		if task.Priority == priority {
			result = append(result, task)
		}
	}
	return result, nil
}

func FilterByStatus(tasks []Task, status string) ([]Task, error) {
	status, err := normalizeChoice(status, StatusNew, StatusWorking, StatusDone)
	if err != nil {
		return nil, fmt.Errorf("недопустимый статус: %w", err)
	}
	var result []Task
	for _, task := range tasks {
		if task.Status == status {
			result = append(result, task)
		}
	}
	return result, nil
}

func CalculateStatistics(tasks []Task) Statistics {
	statistics := Statistics{Total: len(tasks)}
	for _, task := range tasks {
		switch task.Status {
		case StatusNew:
			statistics.New++
		case StatusWorking:
			statistics.Working++
		case StatusDone:
			statistics.Done++
		}
	}
	if statistics.Total > 0 {
		statistics.CompletionPercent = float64(statistics.Done) / float64(statistics.Total) * 100
	}
	return statistics
}
