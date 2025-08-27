//go:build !windows
// +build !windows

package main

// findAllPathOfExileDirectories возвращает пустые списки, так как приложение поддерживает только Windows
func findAllPathOfExileDirectories() ([]string, []string) {
	return []string{}, []string{}
}
