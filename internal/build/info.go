// Package build выводит информацию о версии и сборке бинарного файла.
package build

import "fmt"

// PrintInfo печатает версию, дату сборки и коммит.
func PrintInfo(version, date, commit string) {
	fmt.Println("Build version:", value(version))
	fmt.Println("Build date:", value(date))
	fmt.Println("Build commit:", value(commit))
}

// value возвращает v или "N/A", если строка пустая.
func value(v string) string {
	if v == "" {
		return "N/A"
	}
	return v
}
