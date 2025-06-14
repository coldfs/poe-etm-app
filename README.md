# Every Trade Matters (ETM)
# Path of Exile Trade Notifier

Приложение для отслеживания сообщений о покупке в игре Path of Exile и отправки уведомлений через Telegram.

## Функции

- Автоматический поиск директории Path of Exile в Windows
- Отслеживание сообщений в Client.txt
- Фильтрация сообщений по шаблону "From *: I would like to buy your*"
- Поддерживаются сообщения на английском и русском языках
- Отправка уведомлений через Telegram
- Настраиваемая конфигурация через файл config.ini


## Варианты установки
## Через бота @poe_etm_bot

1. Отредактируйте файл `config.ini` (создается автоматически при первом запуске):

```ini
[API]
ETM_URL = https://etm-bot-server-b74b2ca681a6.herokuapp.com
ETM_TOKEN = xxx
```

2. Замените `ETM_TOKEN` на токен вашего Telegram бота (получите у [@poe_etm_bot](https://t.me/poe_etm_bot))

Токен можно получить перейдя в бота https://t.me/poe_etm_bot и отправим ему команду /start

При жедании можно поднять собственный сервер для бота https://github.com/coldfs/etm-server


## Через своего бота

1. Отредактируйте файл `config.ini` (создается автоматически при первом запуске):

```ini
[Telegram]
; Токен Telegram бота (получить у @BotFather)
BotToken = YOUR_BOT_TOKEN
; ID чата для отправки уведомлений
ChatID = YOUR_CHAT_ID
```

2. Замените `YOUR_BOT_TOKEN` на токен вашего Telegram бота (получите у [@BotFather](https://t.me/BotFather))
3. Замените `YOUR_CHAT_ID` на ID вашего чата в Telegram (узнайте через [@userinfobot](https://t.me/userinfobot))

## Компиляция и запуск

### Требования

- Windows (приложение поддерживает только Windows)
- Go 1.16 или выше

### Сборка

```bash
go build -tags windows
```

### Запуск

```bash
./etm.exe
```


### Мониторинг сообщений

Приложение автоматически отслеживает новые сообщения в файле Client.txt и фильтрует сообщения о покупке.
При обнаружении сообщения о покупке, оно будет отображено в логе и отправлено в Telegram (если настроено).

### Настройка пути к Path of Exile

Если приложение не может автоматически найти директорию Path of Exile, вы можете указать путь вручную в файле config.ini.
После изменения config.ini, перезапустите приложение или дождитесь автоматического обновления настроек.

## Примечания

- Если программа не может найти директорию Path of Exile автоматически, вы можете указать путь вручную при запросе или в файле config.ini
- Приложение поддерживает только Windows
- Для корректной работы необходимо запускать от имени пользователя, который имеет доступ к файлам игры