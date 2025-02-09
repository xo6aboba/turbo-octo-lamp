package main

import (
	"bufio"
	"fmt"
	_ "github.com/go-jose/go-jose/v3/json"
	"github.com/playwright-community/playwright-go"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"strings"
)

var (
	requests    []Request
	isCapturing bool // Флаг для отслеживания состояния перехвата
)

// Структура для хранения информации о запросе
type Request struct {
	Method      string `yaml:"method"`
	URI         string `yaml:"uri"`
	Description string `yaml:"description"`
	Body        string `yaml:"body,omitempty"`
}

func main() {
	fmt.Println("Добро пожаловать в генератор нагрузочных сценариев для Pandora!")

	// Запуск браузера
	startBrowser()

	// Ожидание команд от пользователя
	waitForCommands()

	// Проверка, были ли запросы
	if len(requests) == 0 {
		fmt.Println("Запросы не были зафиксированы.")
		return
	}

}

// Запуск браузера
func startBrowser() {
	// Инициализация Playwright
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("Ошибка при запуске Playwright: %v", err)
	}
	defer pw.Stop()

	// Запуск браузера (например, Chromium)
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false), // Отключаем headless-режим
	})
	if err != nil {
		log.Fatalf("Ошибка при запуске браузера: %v", err)
	}
	defer browser.Close()

	// Открытие новой страницы
	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("Ошибка при создании страницы: %v", err)
	}
	defer page.Close()

	page.SetViewportSize(1920, 960)

	// Логирование всех HTTP-запросов
	page.On("request", func(request playwright.Request) {
		if !isCapturing {
			return // Если перехват не активен, игнорируем запросы
		}

		// Игнорируем запросы, содержащие "sentry" в URL
		if strings.Contains(request.URL(), "sentry") {
			return
		}
		if strings.Contains(request.URL(), "assets") {
			return
		}

		// Логирование запроса
		headers := make(map[string]string)
		for key, value := range request.Headers() {
			headers[key] = value
		}

		// Получение тела запроса
		body, err := request.PostData()
		if err != nil {
			log.Printf("Ошибка при получении тела запроса: %v", err)
			// Если тело запроса недоступно, используем пустую строку
		}

		// Сохранение информации о запросе
		requests = append(requests, Request{
			Method:      request.Method(),
			URI:         request.URL(),
			Description: fmt.Sprintf("Запрос: %s %s", request.Method(), request.URL()),
			Body:        body,
		})

		fmt.Printf("Перехват Запроса: %s %s\n", request.Method(), request.URL())
	})

	// Навигация по URL
	if _, err = page.Goto("https://example.com"); err != nil {
		log.Fatalf("Ошибка при переходе на страницу: %v", err)
	}

	// Ожидание команд от пользователя
	waitForCommands()
}

// Ожидание команд от пользователя
func waitForCommands() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Введите 'start', чтобы начать запись запросов, или 'stop', чтобы завершить запись.")

	for scanner.Scan() {
		command := strings.ToLower(scanner.Text())

		switch command {
		case "start":
			if isCapturing {
				fmt.Println("Запись уже начата.")
			} else {
				isCapturing = true
				fmt.Println("Запись запросов начата.")
			}
		case "stop":
			if !isCapturing {
				fmt.Println("Запись еще не начата.")
			} else {
				isCapturing = false
				fmt.Println("Запись запросов завершена.")
				saveScenario()
				fmt.Println("Сценарий успешно сохранен в файл firstLoadScenario.yaml!")
				os.Exit(1)
				return
			}
		default:
			fmt.Println("Неизвестная команда. Введите 'start' или 'stop'.")
		}

	}
}

// Сохранение сценария в формате YAML
func saveScenario() {
	// Формирование структуры для YAML
	scenario := map[string]interface{}{
		"definition": map[string]interface{}{
			"name":        "Сгенерированный сценарий",
			"description": "Сценарий, сгенерированный автоматически",
			"rps":         "s", // Профиль нагрузки
			"wip":         false,
		},
		"test": map[string]interface{}{
			"requests":      []map[string]interface{}{},
			"preprocessor:": "",
			"mapping":       "",
			"visitor":       "source.data.visitors[rand]",
			"employeeId":    "source.data.employees[rand]",
		},
	}

	// Добавление запросов в сценарий
	for i, req := range requests {
		requestMap := map[string]interface{}{
			"name":        fmt.Sprintf("request_%d", i+1),
			"description": req.Description,
			"uri":         req.URI,
			"user":        "request.profile_employee.preprocessor.visitor.userId",
			"method":      req.Method,
			"body":        req.Body,
		}
		scenario["test"].(map[string]interface{})["requests"] = append(
			scenario["test"].(map[string]interface{})["requests"].([]map[string]interface{}),
			requestMap,
		)
	}

	// Преобразование структуры в YAML
	data, err := yaml.Marshal(scenario)
	if err != nil {
		log.Fatalf("Ошибка при сериализации YAML: %v", err)
	}

	// Сохранение YAML в файл
	file, err := os.Create("firstLoadScenario.yaml")
	if err != nil {
		log.Fatalf("Ошибка при создании файла: %v", err)
	}
	defer file.Close()

	file.Write(data)
}
