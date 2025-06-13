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
		logToUI("⚠️ Предупреждение: не заданы настройки Telegram в config.ini")
		logToUI("Уведомления в Telegram не будут отправляться.")
	}

	var poePath string

	// Проверяем указан ли путь в конфигурации
	if config.CustomPoEPath != "" {
		poePath = config.CustomPoEPath
		logToUI("Используется указанный в config.ini путь: " + poePath)
	} else {
		// Автоматический поиск директории
		poePath, err = findPathOfExileDirectory()
		if err != nil {
			logToUI("Ошибка при поиске директории Path of Exile: " + err.Error())
			logToUI("Пожалуйста, укажите путь к директории Path of Exile вручную в config.ini")

			if runtime.GOOS != "windows" {
				// В консольном режиме запрашиваем путь
				fmt.Println("Пожалуйста, укажите путь к директории Path of Exile вручную:")
				reader := bufio.NewReader(os.Stdin)
				poePath, _ = reader.ReadString('\n')
				poePath = strings.TrimSpace(poePath)

				if poePath == "" {
					fmt.Println("Путь не указан. Выход.")
					return
				}
			} else {
				// Для Windows с трей-иконкой, просто ждем пока пользователь обновит config.ini
				for {
					time.Sleep(5 * time.Second)
					err := loadConfig()
					if err == nil && config.CustomPoEPath != "" {
						poePath = config.CustomPoEPath
						logToUI("Загружен новый путь из config.ini: " + poePath)
						break
					}
				}
			}
		}
	}

	clientLogFilePath := filepath.Join(poePath, "logs", "Client.txt")
	logToUI("Найден файл лога Path of Exile: " + clientLogFilePath)

	err = monitorFile(clientLogFilePath)
	if err != nil {
		logToUI("Ошибка при мониторинге файла: " + err.Error())
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

	// Отладочный вывод
	fmt.Printf("Загруженные настройки:\n")
	fmt.Printf("TelegramBotToken: %s\n", config.TelegramBotToken)
	fmt.Printf("TelegramChatID: %s\n", config.TelegramChatID)
	fmt.Printf("CustomPoEPath: %s\n", config.CustomPoEPath)
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

	// Добавляем параметр parse_mode=MarkdownV2 для поддержки Markdown
	resp, err := http.Get(telegramAPI + "?chat_id=" + config.TelegramChatID + "&text=" + escapedMessage + "&parse_mode=MarkdownV2")
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

	logToUI("Мониторинг файла... (Ctrl+C для выхода)")
	logToUI("Отслеживание сообщений о покупке и отправка в Telegram...")

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

		// Выводим все сообщения в лог
		logToUI(line)

		// Проверяем, соответствует ли сообщение шаблону покупки
		if (strings.Contains(line, "I would like to buy your") || strings.Contains(line, "хочу купить у вас")) && (strings.Contains(line, "@From") || strings.Contains(line, "@От")) {
			// Регулярное выражение для поиска цены, валюты и названия предмета
			// Поддерживает оба языка: английский и русский
			match := regexp.MustCompile(`(?:.*?)(?:I would like to buy your|хочу купить у вас) (.*?) (?:listed for|за) ([\d.]+ (chaos|divine|mirror))(?:.*)`).FindStringSubmatch(line)
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
				}

				// Форматируем сообщение с Markdown-разметкой для жирного шрифта цены
				message := fmt.Sprintf("%s *%s* %s", emoji, price, itemName)
				logToUI("Найдено сообщение о покупке: " + message)

				// Отправляем уведомление в Telegram с поддержкой Markdown
				err := sendTelegramMessage(message)
				if err != nil {
					logToUI("Ошибка отправки в Telegram: " + err.Error())
				} else {
					logToUI("Сообщение успешно отправлено в Telegram")
				}
			} else {
				logToUI("Сообщение не соответствует шаблону: " + line)
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
