//go:build windows
// +build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// findPathOfExileDirectory пытается найти директорию Path of Exile.
// Сначала ищет в реестре Steam, затем проверяет стандартные пути.
func findPathOfExileDirectory() (string, error) {
	// Поиск в реестре Steam
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Valve\Steam`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		steamPath, _, err := k.GetStringValue("InstallPath")
		if err == nil {
			poePath := filepath.Join(steamPath, "steamapps", "common", "Path of Exile")
			if _, err := os.Stat(poePath); err == nil {
				return poePath, nil
			}
		}
	}

	// Если не найдено в реестре, пробуем стандартные пути
	programFilesX86 := os.Getenv("ProgramFiles(x86)")
	if programFilesX86 == "" {
		programFilesX86 = "C:\\Program Files (x86)" // Запасной вариант
	}

	potentialPaths := []string{
		filepath.Join(programFilesX86, "Steam", "steamapps", "common", "Path of Exile"),
		filepath.Join(programFilesX86, "Grinding Gear Games", "Path of Exile"),
		filepath.Join("C:", "SteamLibrary", "steamapps", "common", "Path of Exile"), // Пример альтернативного пути
	}

	for _, p := range potentialPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("директория Path of Exile не найдена")
}
