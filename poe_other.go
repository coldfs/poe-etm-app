//go:build !windows
// +build !windows

package main

import (
	"fmt"
)

// findPathOfExileDirectory возвращает ошибку, так как приложение поддерживает только Windows
func findPathOfExileDirectory() (string, error) {
	return "", fmt.Errorf("автоматический поиск директории Path of Exile поддерживается только для Windows")
}
