package main

import (
	"bufio"
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
	PollInterval     time.Duration
}

var (
	buyMessageRegex = regexp.MustCompile(`From .+: .+buy.+`)
	config          Config
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
		fmt.Println("⚠️ Предупреждение: не заданы настройки Telegram в config.ini")
		fmt.Println("Уведомления в Telegram не будут отправляться.")
	}

	var poePath string

	// Проверяем указан ли путь в конфигурации
	if config.CustomPoEPath != "" {
		poePath = config.CustomPoEPath
		fmt.Printf("Используется указанный в config.ini путь: %s\n", poePath)
	} else {
		// Автоматический поиск директории
		poePath, err = findPathOfExileDirectory()
		if err != nil {
			fmt.Printf("Ошибка при поиске директории Path of Exile: %v\n", err)
			fmt.Println("Пожалуйста, укажите путь к директории Path of Exile вручную:")

			reader := bufio.NewReader(os.Stdin)
			poePath, _ = reader.ReadString('\n')
			poePath = strings.TrimSpace(poePath)

			if poePath == "" {
				fmt.Println("Путь не указан. Выход.")
				return
			}
		}
	}

	clientLogFilePath := filepath.Join(poePath, "Client.txt")
	fmt.Printf("Найден файл лога Path of Exile: %s\n", clientLogFilePath)

	err = monitorFile(clientLogFilePath)
	if err != nil {
		fmt.Printf("Ошибка при мониторинге файла: %v\n", err)
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

	pollInterval := cfg.Section("Settings").Key("PollInterval").MustInt(1)
	config.PollInterval = time.Duration(pollInterval) * time.Second

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
	config.PollInterval = 1 * time.Second

	return nil
}

// sendTelegramMessage отправляет сообщение в Telegram
func sendTelegramMessage(message string) error {
	// Проверяем, указаны ли настройки Telegram
	if config.TelegramBotToken == "YOUR_BOT_TOKEN" || config.TelegramChatID == "YOUR_CHAT_ID" {
		return fmt.Errorf("не указаны настройки Telegram")
	}

	telegramAPI := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.TelegramBotToken)

	// Экранируем сообщение для URL
	escapedMessage := url.QueryEscape(message)

	resp, err := http.Get(telegramAPI + "?chat_id=" + config.TelegramChatID + "&text=" + escapedMessage)
	if err != nil {
		return fmt.Errorf("ошибка при отправке сообщения в Telegram: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка API Telegram, код ответа: %d", resp.StatusCode)
	}

	return nil
}

// monitorFile открывает файл и выводит новые строки.
func monitorFile(filePath string) error {
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

	fmt.Println("Мониторинг файла... (Ctrl+C для выхода)")
	fmt.Println("Отслеживание сообщений о покупке и отправка в Telegram...")

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
		fmt.Println(line)

		// Проверяем, соответствует ли сообщение шаблону покупки
		if buyMessageRegex.MatchString(line) {
			fmt.Println("Найдено сообщение о покупке!")

			// Отправляем уведомление в Telegram
			err := sendTelegramMessage("Новое сообщение о покупке: " + line)
			if err != nil {
				fmt.Printf("Ошибка отправки в Telegram: %v\n", err)
			} else {
				fmt.Println("Сообщение успешно отправлено в Telegram")
			}
		}
	}
}
