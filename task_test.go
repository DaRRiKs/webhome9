package main

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

func mustCreate(t *testing.T, storage TaskStorage, title, description, priority string) Task {
	t.Helper()
	task, err := storage.Create(title, description, priority)
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func TestTaskLifecycle(t *testing.T) {
	var storage TaskStorage = &MemoryTaskStorage{}
	for index, priority := range []string{PriorityLow, PriorityMedium, PriorityHigh} {
		task := mustCreate(t, storage, fmt.Sprintf("Задача %d", index+1), "Описание", priority)
		if task.ID != index+1 || task.Priority != priority || task.Status != StatusNew {
			t.Fatalf("неверные данные созданной задачи: %+v", task)
		}
	}
	for _, status := range []string{StatusWorking, StatusDone, StatusNew, StatusWorking, StatusWorking} {
		if err := storage.SetStatus(2, status); err != nil {
			t.Fatal(err)
		}
		task, err := storage.Get(2)
		if err != nil || task.Status != status {
			t.Fatalf("статус не сохранён: %+v, %v", task, err)
		}
	}
	if err := storage.Update(2, "  Изучить Go  ", " Новое описание ", " ВЫСОКИЙ "); err != nil {
		t.Fatal(err)
	}
	task, err := storage.Get(2)
	want := Task{ID: 2, Title: "Изучить Go", Description: "Новое описание", Priority: PriorityHigh, Status: StatusWorking}
	if err != nil || task != want {
		t.Fatalf("редактирование: получено %+v, %v; ожидалось %+v", task, err, want)
	}
	list := storage.All()
	list[0].Title = "Изменённая копия"
	task, err = storage.Get(1)
	if err != nil || task.Title != "Задача 1" {
		t.Fatalf("изменение копии списка затронуло хранилище: %+v, %v", task, err)
	}
	if err := storage.Delete(2); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.Get(2); err == nil {
		t.Fatal("удалённая задача найдена")
	}
	list = storage.All()
	if len(list) != 2 || list[0].ID != 1 || list[1].ID != 3 {
		t.Fatalf("неверный список после удаления: %+v", list)
	}
	if task := mustCreate(t, storage, "Четвёртая", "", "низкий"); task.ID != 4 {
		t.Fatalf("повторно использован ID: %d", task.ID)
	}
	for _, id := range []int{1, 4, 3} {
		if err := storage.Delete(id); err != nil {
			t.Fatal(err)
		}
	}
	if len(storage.All()) != 0 {
		t.Fatal("хранилище не опустело после удаления всех задач")
	}
	if task := mustCreate(t, storage, "Пятая", "", PriorityMedium); task.ID != 5 {
		t.Fatalf("ID сбросился после удаления всех задач: %d", task.ID)
	}
}

func TestInvalidCreationDoesNotChangeStorage(t *testing.T) {
	storage := &MemoryTaskStorage{}
	for _, input := range []struct{ title, priority string }{
		{"", PriorityLow},
		{" \t ", PriorityMedium},
		{"Название", ""},
		{"Название", "Срочный"},
	} {
		if _, err := storage.Create(input.title, "", input.priority); err == nil {
			t.Fatalf("приняты недопустимые данные: %+v", input)
		}
	}
	if len(storage.All()) != 0 {
		t.Fatal("неудачное создание изменило хранилище")
	}
	task := mustCreate(t, storage, "  Задача  ", "", "сРеДнИй")
	if task.ID != 1 || task.Title != "Задача" || task.Priority != PriorityMedium {
		t.Fatalf("неверные данные после повторного создания: %+v", task)
	}
}

func TestRejectedChangesLeaveTaskUnchanged(t *testing.T) {
	storage := &MemoryTaskStorage{}
	original := mustCreate(t, storage, "Исходное название", "Исходное описание", PriorityLow)
	actions := []func() error{
		func() error {
			return storage.Update(original.ID, " \t ", "Другое описание", PriorityHigh)
		},
		func() error {
			return storage.Update(original.ID, "Другое название", "", "Срочный")
		},
		func() error { return storage.SetStatus(original.ID, "Отложена") },
		func() error { return storage.SetStatus(original.ID, "") },
	}
	for index, action := range actions {
		if err := action(); err == nil {
			t.Fatalf("операция %d не отклонила ошибочные данные", index)
		}
		task, err := storage.Get(original.ID)
		if err != nil || task != original {
			t.Fatalf("операция %d повредила задачу: %+v, %v", index, task, err)
		}
	}
}

func TestMissingTask(t *testing.T) {
	storage := &MemoryTaskStorage{}
	mustCreate(t, storage, "Задача", "", PriorityLow)
	for _, id := range []int{-1, 0, 999, 999} {
		_, getErr := storage.Get(id)
		for _, err := range []error{
			getErr,
			storage.Update(id, "Название", "", PriorityHigh),
			storage.SetStatus(id, StatusDone),
			storage.Delete(id),
		} {
			if err == nil {
				t.Fatalf("операция с отсутствующим ID %d не вернула ошибку", id)
			}
		}
	}
	if len(storage.All()) != 1 {
		t.Fatal("ошибочные операции изменили количество задач")
	}
}

func TestSearchAndFilters(t *testing.T) {
	tasks := []Task{
		{ID: 1, Title: "Изучить Go", Priority: PriorityLow, Status: StatusNew},
		{ID: 2, Title: "Изучить Golang", Priority: PriorityMedium, Status: StatusWorking},
		{ID: 3, Title: "Купить книгу", Priority: PriorityHigh, Status: StatusDone},
	}
	tests := []struct {
		name    string
		search  func([]Task, string) ([]Task, error)
		value   string
		wantIDs []int
		wantErr bool
	}{
		{"часть названия", SearchByTitle, " gO ", []int{1, 2}, false},
		{"кириллица", SearchByTitle, "ИЗУЧИТЬ", []int{1, 2}, false},
		{"нет совпадений", SearchByTitle, "несуществующая", nil, false},
		{"пустой запрос", SearchByTitle, " \t ", nil, true},
		{"низкий приоритет", FilterByPriority, "низкий", []int{1}, false},
		{"средний приоритет", FilterByPriority, PriorityMedium, []int{2}, false},
		{"высокий приоритет", FilterByPriority, PriorityHigh, []int{3}, false},
		{"ошибочный приоритет", FilterByPriority, "Срочный", nil, true},
		{"новые", FilterByStatus, StatusNew, []int{1}, false},
		{"в работе", FilterByStatus, " В РАБОТЕ ", []int{2}, false},
		{"завершённые", FilterByStatus, StatusDone, []int{3}, false},
		{"ошибочный статус", FilterByStatus, "Отложена", nil, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.search(tasks, test.value)
			if (err != nil) != test.wantErr {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			var ids []int
			for _, task := range result {
				ids = append(ids, task.ID)
			}
			if !reflect.DeepEqual(ids, test.wantIDs) {
				t.Fatalf("получены ID %v, ожидались %v", ids, test.wantIDs)
			}
		})
	}
}

func TestStatistics(t *testing.T) {
	tests := []struct {
		name     string
		statuses []string
		want     Statistics
	}{
		{"нет задач", nil, Statistics{}},
		{"одна новая", []string{StatusNew}, Statistics{Total: 1, New: 1}},
		{"все завершены", []string{StatusDone, StatusDone}, Statistics{Total: 2, Done: 2, CompletionPercent: 100}},
		{"дробный процент", []string{StatusNew, StatusWorking, StatusDone}, Statistics{Total: 3, New: 1, Working: 1, Done: 1, CompletionPercent: 100.0 / 3}},
		{"пример задания", []string{StatusNew, StatusNew, StatusWorking, StatusDone, StatusDone}, Statistics{Total: 5, New: 2, Working: 1, Done: 2, CompletionPercent: 40}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var tasks []Task
			for _, status := range test.statuses {
				tasks = append(tasks, Task{Status: status})
			}
			got := CalculateStatistics(tasks)
			if got.Total != test.want.Total || got.New != test.want.New || got.Working != test.want.Working || got.Done != test.want.Done || math.Abs(got.CompletionPercent-test.want.CompletionPercent) > 0.000001 {
				t.Fatalf("получено %+v, ожидалось %+v", got, test.want)
			}
		})
	}
}
