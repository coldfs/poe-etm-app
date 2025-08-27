//go:build windows
// +build windows

package main

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// findAllPathOfExileDirectories ищет все установленные версии Path of Exile (1 и 2)
func findAllPathOfExileDirectories() ([]string, []string) {
	var poePaths []string
	var poe2Paths []string
	
	// Поиск в реестре Steam
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Valve\Steam`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		steamPath, _, err := k.GetStringValue("InstallPath")
		if err == nil {
			// Проверяем Path of Exile 1
			poePath := filepath.Join(steamPath, "steamapps", "common", "Path of Exile")
			if _, err := os.Stat(poePath); err == nil {
				poePaths = append(poePaths, poePath)
			}
			// Проверяем Path of Exile 2
			poe2Path := filepath.Join(steamPath, "steamapps", "common", "Path of Exile 2")
			if _, err := os.Stat(poe2Path); err == nil {
				poe2Paths = append(poe2Paths, poe2Path)
			}
		}
	}

	// Стандартные пути установки
	programFilesX86 := os.Getenv("ProgramFiles(x86)")
	if programFilesX86 == "" {
		programFilesX86 = "C:\\Program Files (x86)"
	}
	
	programFiles := os.Getenv("ProgramFiles")
	if programFiles == "" {
		programFiles = "C:\\Program Files"
	}

	// Потенциальные пути для Path of Exile 1
	potentialPoE1Paths := []string{
		filepath.Join(programFilesX86, "Steam", "steamapps", "common", "Path of Exile"),
		filepath.Join(programFilesX86, "Grinding Gear Games", "Path of Exile"),
		filepath.Join(programFiles, "Grinding Gear Games", "Path of Exile"),
		filepath.Join("C:", "Games", "Path of Exile"),
		filepath.Join("C:", "SteamLibrary", "steamapps", "common", "Path of Exile"),
		filepath.Join("D:", "SteamLibrary", "steamapps", "common", "Path of Exile"),
		filepath.Join("D:", "Games", "Path of Exile"),
	}

	// Потенциальные пути для Path of Exile 2
	potentialPoE2Paths := []string{
		filepath.Join(programFilesX86, "Steam", "steamapps", "common", "Path of Exile 2"),
		filepath.Join(programFilesX86, "Grinding Gear Games", "Path of Exile 2"),
		filepath.Join(programFiles, "Grinding Gear Games", "Path of Exile 2"),
		filepath.Join("C:", "Games", "Path of Exile 2"),
		filepath.Join("C:", "SteamLibrary", "steamapps", "common", "Path of Exile 2"),
		filepath.Join("D:", "SteamLibrary", "steamapps", "common", "Path of Exile 2"),
		filepath.Join("D:", "Games", "Path of Exile 2"),
	}

	// Проверяем пути для PoE 1
	for _, p := range potentialPoE1Paths {
		if _, err := os.Stat(p); err == nil {
			// Проверяем, что путь не дублируется
			duplicate := false
			for _, existing := range poePaths {
				if existing == p {
					duplicate = true
					break
				}
			}
			if !duplicate {
				poePaths = append(poePaths, p)
			}
		}
	}

	// Проверяем пути для PoE 2
	for _, p := range potentialPoE2Paths {
		if _, err := os.Stat(p); err == nil {
			// Проверяем, что путь не дублируется
			duplicate := false
			for _, existing := range poe2Paths {
				if existing == p {
					duplicate = true
					break
				}
			}
			if !duplicate {
				poe2Paths = append(poe2Paths, p)
			}
		}
	}

	return poePaths, poe2Paths
}
