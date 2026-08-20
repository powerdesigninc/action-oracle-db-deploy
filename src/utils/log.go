package utils

import "fmt"

const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
)

func Red(s string) string    { return ansiRed + s + ansiReset }
func Green(s string) string  { return ansiGreen + s + ansiReset }
func Yellow(s string) string { return ansiYellow + s + ansiReset }

func PrintError(err interface{}) {
	fmt.Println(Red(fmt.Sprintf("Error: %s", err)))
}

func PrintWarning(a ...interface{}) {
	fmt.Println(Yellow(fmt.Sprint(a...)))
}
