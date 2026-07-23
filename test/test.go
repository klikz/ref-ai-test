package main

import (
	"fmt"
	"time"
)

func main() {

	paddedCount := fmt.Sprintf("%05d", 1)
	serial := generateSerial("A1FA01", 1, paddedCount)

	fmt.Println(serial)
}

func generateSerial(modelAndVersion string, line int, code string) string {
	serial := ""

	serial += modelAndVersion
	serial += fmt.Sprintf("%d", line) // This adds line as a plain number

	currentYear := time.Now().Year()
	year := string(rune('A' + (currentYear - 2023)))
	serial += year

	month := GetMonthValueAsString()
	serial += month

	serial += code

	fmt.Println("year: ", year)
	fmt.Println("month: ", month)
	fmt.Println("code: ", code)

	return serial
}

func GetMonthValueAsString() string {
	monthValue := time.Now().Month()

	if monthValue < 10 {
		return fmt.Sprintf("%d", monthValue)
	}

	alphabetChar := 'A' + rune(monthValue-10)
	return string(alphabetChar)
}
