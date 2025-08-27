package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"gopkg.in/ini.v1" // Для работы с ini-файлами
)

// Структура с настройками приложения
type Config struct {
	TelegramBotToken string
	TelegramChatID   string
	CustomPoEPath    string
	CustomPoE2Path   string
	PollInterval     time.Duration
	ETMURL           string
	ETMToken         string
}

var (
	config Config
)

func main() {
	// Проверяем операционную систему
	if runtime.GOOS != "windows" {
		fmt.Println("Это приложение работает только в Windows!")
		return
	}

	// Загружаем конфигурацию
	err := loadConfig()
	if err != nil {
		fmt.Printf("Ошибка при загрузке конфигурации: %v\n", err)
		fmt.Println("Будут использованы значения по умолчанию.")
	}

	// Проверяем настройки Telegram
	if config.TelegramBotToken == "YOUR_BOT_TOKEN" || config.TelegramChatID == "YOUR_CHAT_ID" {
		logToUI("⚠️ Предупреждение: не заданы настройки Telegram в config.ini")
		logToUI("Уведомления в Telegram не будут отправляться.")
	}

	var pathsToMonitor []string

	// Если указаны пользовательские пути, используем их
	if config.CustomPoEPath != "" {
		clientLogPath := filepath.Join(config.CustomPoEPath, "logs", "Client.txt")
		if _, err := os.Stat(clientLogPath); err == nil {
			pathsToMonitor = append(pathsToMonitor, clientLogPath)
			logToUI("Используется указанный в config.ini путь PoE 1: " + config.CustomPoEPath)
		} else {
			logToUI("Предупреждение: не найден Client.txt по пути PoE 1: " + config.CustomPoEPath)
		}
	}

	if config.CustomPoE2Path != "" {
		clientLogPath := filepath.Join(config.CustomPoE2Path, "logs", "Client.txt")
		if _, err := os.Stat(clientLogPath); err == nil {
			pathsToMonitor = append(pathsToMonitor, clientLogPath)
			logToUI("Используется указанный в config.ini путь PoE 2: " + config.CustomPoE2Path)
		} else {
			logToUI("Предупреждение: не найден Client.txt по пути PoE 2: " + config.CustomPoE2Path)
		}
	}

	// Если пути не указаны, ищем автоматически
	if len(pathsToMonitor) == 0 {
		logToUI("Автоматический поиск директорий Path of Exile...")
		poePaths, poe2Paths := findAllPathOfExileDirectories()
		
		// Добавляем пути PoE 1
		for _, path := range poePaths {
			clientLogPath := filepath.Join(path, "logs", "Client.txt")
			if _, err := os.Stat(clientLogPath); err == nil {
				pathsToMonitor = append(pathsToMonitor, clientLogPath)
				logToUI("Найден Path of Exile: " + path)
			}
		}
		
		// Добавляем пути PoE 2
		for _, path := range poe2Paths {
			clientLogPath := filepath.Join(path, "logs", "Client.txt")
			if _, err := os.Stat(clientLogPath); err == nil {
				pathsToMonitor = append(pathsToMonitor, clientLogPath)
				logToUI("Найден Path of Exile 2: " + path)
			}
		}
		
		if len(pathsToMonitor) == 0 {
			logToUI("Ошибка: не найдено ни одной установленной версии Path of Exile")
			logToUI("Пожалуйста, укажите путь к директории Path of Exile вручную в config.ini")
			
			// Ждем пока пользователь обновит config.ini
			for {
				time.Sleep(5 * time.Second)
				err := loadConfig()
				if err == nil && (config.CustomPoEPath != "" || config.CustomPoE2Path != "") {
					logToUI("Загружены новые пути из config.ini")
					main() // Рекурсивно вызываем main для переинициализации
					return
				}
			}
		}
	}

	// Запускаем мониторинг всех найденных файлов
	logToUI(fmt.Sprintf("Начинаем мониторинг %d файл(ов) Client.txt", len(pathsToMonitor)))
	
	errChan := make(chan error)
	for _, path := range pathsToMonitor {
		go func(p string) {
			logToUI("Мониторинг: " + p)
			err := monitorFile(p)
			if err != nil {
				errChan <- fmt.Errorf("ошибка мониторинга %s: %w", p, err)
			}
		}(path)
	}
	
	// Ожидаем ошибок (программа будет работать бесконечно, если все в порядке)
	for err := range errChan {
		logToUI(err.Error())
	}
}

// loadConfig загружает настройки из файла config.ini
func loadConfig() error {
	// Проверяем существование файла
	if _, err := os.Stat("config.ini"); os.IsNotExist(err) {
		// Файл не существует, создаем с настройками по умолчанию
		return createDefaultConfig()
	}

	// Загружаем файл конфигурации
	cfg, err := ini.Load("config.ini")
	if err != nil {
		return fmt.Errorf("не удалось загрузить config.ini: %w", err)
	}

	// Загружаем настройки
	config.TelegramBotToken = cfg.Section("Telegram").Key("BotToken").String()
	config.TelegramChatID = cfg.Section("Telegram").Key("ChatID").String()
	config.CustomPoEPath = cfg.Section("PathOfExile").Key("CustomPath").String()
	config.CustomPoE2Path = cfg.Section("PathOfExile2").Key("CustomPath").String()

	// Загружаем настройки API с подробным логированием
	apiSection := cfg.Section("API")
	config.ETMURL = apiSection.Key("ETM_URL").String()
	config.ETMToken = apiSection.Key("ETM_TOKEN").String()

	pollInterval := cfg.Section("Settings").Key("PollInterval").MustInt(1)
	config.PollInterval = time.Duration(pollInterval) * time.Second

	// Отладочный вывод
	fmt.Printf("Загруженные настройки:\n")
	fmt.Printf("TelegramBotToken: %s\n", config.TelegramBotToken)
	fmt.Printf("TelegramChatID: %s\n", config.TelegramChatID)
	fmt.Printf("CustomPoEPath: %s\n", config.CustomPoEPath)
	fmt.Printf("ETM_URL: %s\n", config.ETMURL)
	fmt.Printf("ETM_TOKEN: %s\n", config.ETMToken)
	fmt.Printf("PollInterval: %v\n", config.PollInterval)

	return nil
}

// createDefaultConfig создает файл конфигурации с настройками по умолчанию
func createDefaultConfig() error {
	// Создаем структуру для INI файла
	cfg := ini.Empty()

	// Секция Telegram
	telegramSection, _ := cfg.NewSection("Telegram")
	telegramSection.NewKey("BotToken", "YOUR_BOT_TOKEN")
	telegramSection.NewKey("ChatID", "YOUR_CHAT_ID")

	// Секция PathOfExile
	poeSection, _ := cfg.NewSection("PathOfExile")
	poeSection.NewKey("CustomPath", "")
	
	// Секция PathOfExile2
	poe2Section, _ := cfg.NewSection("PathOfExile2")
	poe2Section.NewKey("CustomPath", "")

	// Секция API
	apiSection, _ := cfg.NewSection("API")
	apiSection.NewKey("ETM_URL", "")
	apiSection.NewKey("ETM_TOKEN", "")

	// Секция Settings
	settingsSection, _ := cfg.NewSection("Settings")
	settingsSection.NewKey("PollInterval", "1")

	// Сохраняем INI файл
	if err := cfg.SaveTo("config.ini"); err != nil {
		return fmt.Errorf("не удалось создать config.ini: %w", err)
	}

	// Устанавливаем настройки по умолчанию
	config.TelegramBotToken = "YOUR_BOT_TOKEN"
	config.TelegramChatID = "YOUR_CHAT_ID"
	config.CustomPoEPath = ""
	config.CustomPoE2Path = ""
	config.ETMURL = "https://etm-bot-server-b74b2ca681a6.herokuapp.com"
	config.ETMToken = ""
	config.PollInterval = 1 * time.Second

	return nil
}

// sendTelegramMessage отправляет сообщение в Telegram
func sendTelegramMessage(message string) error {
	// Проверяем, указаны ли настройки Telegram
	if config.TelegramBotToken == "YOUR_BOT_TOKEN" || config.TelegramChatID == "YOUR_CHAT_ID" {
		logToUI("Debug - Telegram не настроен: BotToken или ChatID не заданы в config.ini")
		return fmt.Errorf("не указаны настройки Telegram")
	}

	telegramAPI := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.TelegramBotToken)
	escapedMessage := url.QueryEscape(message)
	fullURL := telegramAPI + "?chat_id=" + config.TelegramChatID + "&text=" + escapedMessage + "&parse_mode=MarkdownV2"

	logToUI("Debug - Отправка запроса в Telegram")
	logToUI("Debug - URL: " + fullURL)

	resp, err := http.Get(fullURL)
	if err != nil {
		logToUI("Debug - Ошибка при отправке запроса в Telegram: " + err.Error())
		return fmt.Errorf("ошибка при отправке сообщения в Telegram: %w", err)
	}
	defer resp.Body.Close()

	// Читаем тело ответа для отладки
	body, _ := io.ReadAll(resp.Body)
	logToUI(fmt.Sprintf("Debug - Ответ Telegram (код %d): %s", resp.StatusCode, string(body)))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка API Telegram, код ответа: %d", resp.StatusCode)
	}

	return nil
}

// sendMessageViaAPI отправляет сообщение через API
func sendMessageViaAPI(message string) error {
	if config.ETMURL == "" || config.ETMToken == "" {
		logToUI("Debug - API не настроен: ETM_URL или ETM_TOKEN не заданы в config.ini")
		return fmt.Errorf("не заданы настройки API в config.ini")
	}

	// Формируем JSON для отправки
	jsonData := map[string]string{
		"token":   config.ETMToken,
		"message": message,
	}
	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		logToUI("Debug - Ошибка при формировании JSON: " + err.Error())
		return fmt.Errorf("ошибка при формировании JSON: %w", err)
	}

	apiURL := config.ETMURL + "/message"
	logToUI("Debug - Отправка POST запроса на " + apiURL)
	logToUI("Debug - Тело запроса: " + string(jsonBytes))

	// Отправляем POST запрос
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		logToUI("Debug - Ошибка при отправке POST запроса: " + err.Error())
		return fmt.Errorf("ошибка при отправке POST запроса: %w", err)
	}
	defer resp.Body.Close()

	// Читаем тело ответа для отладки
	body, _ := io.ReadAll(resp.Body)
	logToUI(fmt.Sprintf("Debug - Ответ сервера (код %d): %s", resp.StatusCode, string(body)))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка API, код ответа: %d", resp.StatusCode)
	}

	return nil
}

// sendMessage отправляет сообщение через доступные каналы
func sendMessage(message string) error {
	logToUI("Debug - Попытка отправки сообщения: " + message)

	// Пробуем отправить через API, если заданы переменные окружения
	logToUI("Debug - Пробуем отправить через API...")
	err := sendMessageViaAPI(message)
	if err == nil {
		logToUI("Debug - Сообщение успешно отправлено через API")
		return nil
	}
	logToUI("Debug - Не удалось отправить через API: " + err.Error())

	// Если API недоступен, пробуем отправить через Telegram
	logToUI("Debug - Пробуем отправить через Telegram...")
	err = sendTelegramMessage(message)
	if err != nil {
		logToUI("Debug - Не удалось отправить через Telegram: " + err.Error())
		return err
	}
	logToUI("Debug - Сообщение успешно отправлено через Telegram")
	return nil
}

// monitorFile открывает файл и выводит новые строки.
func monitorFile(filePath string) error {
	// Определяем из какой игры лог
	gameVersion := "PoE"
	if strings.Contains(filePath, "Path of Exile 2") {
		gameVersion = "PoE 2"
	}
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл %s: %w", filePath, err)
	}
	defer file.Close()

	// Переходим в конец файла при первом открытии
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("не удалось переместиться в конец файла: %w", err)
	}

	reader := bufio.NewReader(file)

	logToUI(fmt.Sprintf("[%s] Мониторинг файла... (Ctrl+C для выхода)", gameVersion))
	logToUI(fmt.Sprintf("[%s] Отслеживание сообщений о покупке и отправка в Telegram...", gameVersion))

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Достигнут конец файла, ждем новых данных
				time.Sleep(config.PollInterval)
				continue
			}
			return fmt.Errorf("ошибка чтения файла: %w", err)
		}

		// Обрезаем пробельные символы в конце строки
		line = strings.TrimRight(line, "\r\n")

		// Выводим все сообщения в лог с префиксом игры
		logToUI(fmt.Sprintf("[%s] %s", gameVersion, line))

		// Проверяем, соответствует ли сообщение шаблону покупки
		if (strings.Contains(line, "I would like to buy your") || strings.Contains(line, "хочу купить у вас")) && (strings.Contains(line, "@From") || strings.Contains(line, "@От")) {
			// Регулярное выражение для поиска цены, валюты и названия предмета
			// Поддерживает оба языка: английский и русский
			match := regexp.MustCompile(`(?:.*?)(?:I would like to buy your|хочу купить у вас) (.*?) (?:listed for|за) ([\d.]+ (chaos|divine|mirror|exalted))(?:.*)`).FindStringSubmatch(line)
			if len(match) > 0 {
				itemName := strings.TrimSpace(match[1])
				price := strings.TrimSpace(match[2])

				// Определяем эмодзи в зависимости от валюты
				emoji := "💰" // по умолчанию
				if strings.Contains(price, "divine") {
					emoji = "💎" // для divine
				} else if strings.Contains(price, "chaos") {
					emoji = "🪙" // для chaos
				} else if strings.Contains(price, "mirror") {
					emoji = "🪞" // для mirror
				} else if strings.Contains(price, "exalted") {
					emoji = "✨" // для exalted
				}

				// Форматируем сообщение с Markdown-разметкой для жирного шрифта цены
				message := fmt.Sprintf("%s *%s* %s", emoji, price, itemName)
				logToUI(fmt.Sprintf("[%s] Найдено сообщение о покупке: %s", gameVersion, message))

				// Отправляем уведомление через доступные каналы
				err := sendMessage(message)
				if err != nil {
					logToUI(fmt.Sprintf("[%s] Ошибка отправки сообщения: %s", gameVersion, err.Error()))
				} else {
					logToUI(fmt.Sprintf("[%s] Сообщение успешно отправлено", gameVersion))
				}
			} else {
				logToUI(fmt.Sprintf("[%s] Сообщение не соответствует шаблону: %s", gameVersion, line))
			}
		}
	}
}

// logToUI выводит сообщение в лог (консоль или UI)
func logToUI(message string) {
	// Выводим в консоль в любом случае
	fmt.Println(message)

	// Если запущен в Windows с UI, отправляем в окно
	// if runtime.GOOS == "windows" {
	// 	// В файле tray_windows.go определена функция AddLogMessage
	// 	AddLogMessage(message)
	// }
}
