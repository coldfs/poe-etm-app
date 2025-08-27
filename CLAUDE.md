# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is "Every Trade Matters (ETM)", a Windows-only application that monitors trade messages in both Path of Exile and Path of Exile 2, sending notifications via Telegram. The application is written in Go and supports simultaneous monitoring of both game versions.

## Build and Development Commands

```bash
# Build for Windows (required for proper Windows features)
go build -tags windows

# Run the application
./etm.exe
```

## Architecture Overview

### Core Components

- **main.go**: Main application logic with parallel file monitoring, config loading, and message processing for both game versions
- **poe_windows.go**: Windows-specific directory discovery for both PoE 1 and PoE 2 using registry and standard paths  
- **poe_other.go**: Non-Windows stub (returns empty lists as app is Windows-only)

### Key Functionality

1. **Path Discovery**: Automatically finds both Path of Exile and Path of Exile 2 installations via Windows registry (Steam) or standard paths
2. **Parallel File Monitoring**: Simultaneously monitors multiple `Client.txt` log files from both game versions using goroutines
3. **Message Processing**: Filters trade messages using regex patterns for English/Russian languages with game version prefixes [PoE] or [PoE 2]
4. **Notification Delivery**: Supports two channels:
   - Direct Telegram Bot API
   - ETM server API (https://etm-bot-server-b74b2ca681a6.herokuapp.com)

### Configuration System

The app uses `config.ini` with auto-generation of default config:

```ini
[Telegram]
BotToken = YOUR_BOT_TOKEN
ChatID = YOUR_CHAT_ID

[API] 
ETM_URL = https://etm-bot-server-b74b2ca681a6.herokuapp.com
ETM_TOKEN = xxx

[PathOfExile]
CustomPath = 

[PathOfExile2]
CustomPath =

[Settings]
PollInterval = 1
```

### Message Pattern Detection

The application detects trade messages matching patterns:
- English: `"I would like to buy your"` with `"@From"`
- Russian: `"хочу купить у вас"` with `"@От"`

Extracts item name, price, and currency (chaos/divine/mirror) with appropriate emoji formatting.

## Dependencies

Key Go modules:
- `gopkg.in/ini.v1` - INI configuration file handling
- `golang.org/x/sys/windows/registry` - Windows registry access for Steam path discovery
- Standard library for HTTP, file I/O, and regex

## Platform Requirements

- Windows only (registry access, Path of Exile Windows paths)
- Go 1.16+ for build
- Path of Exile installation for log file monitoring