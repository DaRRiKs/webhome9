package main

import (
	"bytes"
	"strings"
	"testing"
)

func runConsole(t *testing.T, storage TaskStorage, input string) string {
	t.Helper()
	var output bytes.Buffer
	if err := NewConsole(storage, strings.NewReader(input), &output).Run(); err != nil {
		t.Fatal(err)
	}
	return output.String()
}

func TestConsoleWorkflow(t *testing.T) {
	storage := &MemoryTaskStorage{}
	input := strings.Join([]string{
		"7", "2",
		"1", "Изучить Go", "Структуры", "Низкий",
		"1", "Прочитать книгу", "Глава 9", "Средний",
		"1", "Сдать задание", "Модуль 09", "Высокий",
		"2",
		"3", "1", "В работе",
		"3", "2", "Завершена",
		"3", "3", "Новая",
		"7",
		"4", "3", "Проверить задание", "Проверить ввод", "Средний",
		"2",
		"6", "1", "ИЗУЧИТЬ",
		"6", "2", "Средний",
		"6", "3", "Завершена",
		"5", "1",
		"7",
		"5", "2",
		"5", "3",
		"7", "2",
		"1", "Новая после удаления", "", "Высокий",
		"2", "0",
	}, "\n")
	output := runConsole(t, storage, input)
	for _, expected := range []string{
		"Задача #1 успешно создана!",
		"Задача #2 успешно создана!",
		"Задача #3 успешно создана!",
		"1 | Изучить Go | Низкий | Новая",
		"2 | Прочитать книгу | Средний | Новая",
		"3 | Сдать задание | Высокий | Новая",
		"Статус задачи #1 изменён.",
		"Статус задачи #2 изменён.",
		"Статус задачи #3 изменён.",
		"Всего задач: 3\nНовых: 1\nВ работе: 1\nЗавершённых: 1\nПроцент выполнения: 33.33%",
		"Задача #3 обновлена.",
		"3 | Проверить задание | Средний | Новая",
		"Задача #1 удалена.",
		"Всего задач: 2\nНовых: 1\nВ работе: 0\nЗавершённых: 1\nПроцент выполнения: 50.00%",
		"Всего задач: 0\nНовых: 0\nВ работе: 0\nЗавершённых: 0\nПроцент выполнения: 0.00%",
		"Задачи не найдены.",
		"4 | Новая после удаления | Высокий | Новая",
		"До свидания!",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("в выводе отсутствует %q", expected)
		}
	}
	if strings.Contains(output, "Ошибка:") {
		t.Fatalf("ошибка в корректном сценарии:\n%s", output)
	}
	if tasks := storage.All(); len(tasks) != 1 || tasks[0].ID != 4 {
		t.Fatalf("неверное итоговое состояние: %+v", tasks)
	}
}

func TestConsoleSearchResults(t *testing.T) {
	storage := &MemoryTaskStorage{}
	mustCreate(t, storage, "Первая задача", "", PriorityLow)
	mustCreate(t, storage, "Вторая задача", "", PriorityHigh)
	if err := storage.SetStatus(2, StatusDone); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{"6\n1\nВТОРАЯ\n0\n", "6\n2\nВысокий\n0\n", "6\n3\nЗавершена\n0\n"} {
		output := runConsole(t, storage, input)
		if !strings.Contains(output, "2 | Вторая задача | Высокий | Завершена") || strings.Contains(output, "1 | Первая задача") {
			t.Fatalf("неверный результат поиска:\n%s", output)
		}
	}
	output := runConsole(t, storage, "6\n1\nнет совпадений\n0\n")
	if !strings.Contains(output, "Задачи не найдены.") {
		t.Fatal("отсутствует сообщение о пустом результате")
	}
}

func TestConsoleInvalidInput(t *testing.T) {
	storage := &MemoryTaskStorage{}
	original := mustCreate(t, storage, "Исходная", "Описание", PriorityLow)
	input := strings.Join([]string{
		"ошибка", "9", "",
		"1", "   ",
		"1", "Название", "", "Срочный",
		"3", "abc",
		"3", "0",
		"4", "-1",
		"5", "1.5",
		"5", "9999999999999999999999999999999999",
		"3", "999",
		"4", "999",
		"5", "999",
		"3", "1", "Отложена",
		"4", "1", "",
		"4", "1", "Изменённая", "Другое описание", "Ошибка",
		"6", "9",
		"6", "1", "",
		"6", "2", "Ошибка",
		"6", "3", "Ошибка",
		"2", "0",
	}, "\n")
	output := runConsole(t, storage, input)
	if count := strings.Count(output, "Ошибка:"); count != 20 {
		t.Fatalf("получено %d ошибок вместо 20:\n%s", count, output)
	}
	for _, expected := range []string{
		"название задачи не может быть пустым",
		"недопустимый приоритет",
		"недопустимый статус",
		"ID должен быть положительным целым числом",
		"задача с ID 999 не найдена",
		"выберите способ поиска от 1 до 3",
		"поисковый запрос не может быть пустым",
		"1 | Исходная | Низкий | Новая",
		"До свидания!",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("в выводе отсутствует %q", expected)
		}
	}
	task, err := storage.Get(original.ID)
	if err != nil || task != original || len(storage.All()) != 1 {
		t.Fatalf("ошибочный ввод повредил хранилище: %+v, %v", task, err)
	}
}

func TestConsoleEndOfInput(t *testing.T) {
	for _, input := range []string{
		"", "0", "1", "1\nНазвание", "1\nНазвание\nОписание\n",
		"3\n", "3\n1\n", "4\n1\nНовое название\n", "4\n1\nНовое\nОписание\n",
		"5\n", "6\n", "6\n1\n", "6\n2\n", "6\n3\n",
	} {
		t.Run(input, func(t *testing.T) {
			storage := &MemoryTaskStorage{}
			original := mustCreate(t, storage, "Исходная", "Описание", PriorityLow)
			output := runConsole(t, storage, input)
			if strings.Contains(output, "Ошибка:") {
				t.Fatalf("ошибка при завершении ввода:\n%s", output)
			}
			task, err := storage.Get(original.ID)
			if err != nil || task != original || len(storage.All()) != 1 {
				t.Fatalf("незавершённая операция повредила хранилище: %+v, %v", task, err)
			}
		})
	}
}

func TestConsoleLineEndingsAndLongInput(t *testing.T) {
	storage := &MemoryTaskStorage{}
	title := strings.Repeat("Я", 70000)
	runConsole(t, storage, "1\r\n"+title+"\r\n\r\nВысокий\r\n0")
	tasks := storage.All()
	if len(tasks) != 1 || tasks[0].Title != title {
		t.Fatal("длинный ввод или переводы строк Windows обработаны неверно")
	}
}

func TestConsoleFreshStorage(t *testing.T) {
	runConsole(t, &MemoryTaskStorage{}, "1\nЗадача\nОписание\nНизкий\n0\n")
	output := runConsole(t, &MemoryTaskStorage{}, "2\n7\n0\n")
	if !strings.Contains(output, "Задачи не найдены.") || !strings.Contains(output, "Всего задач: 0") {
		t.Fatal("новое хранилище содержит данные предыдущего запуска")
	}
}
