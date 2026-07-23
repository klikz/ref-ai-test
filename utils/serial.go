package utils

import (
	"fmt"
	"math/rand/v2"
	"time"
)

func GenerateSerial(prefix string, count int) string {
	currentYear := time.Now().Year()
	year := string(rune('A' + (currentYear - 2023)))

	monthValue := time.Now().Month()
	month := ""
	if monthValue < 10 {
		month = fmt.Sprintf("%d", monthValue)
	} else {
		month = string('A' + rune(monthValue-10))
	}

	dayValue := time.Now().Day()
	day := ""
	if dayValue < 10 {
		day = fmt.Sprintf("%d", dayValue)
	} else {
		day = string('A' + rune(dayValue-10))
	}

	counter := fmt.Sprintf("%04d", count)

	letterRunes := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	randomLetter := string(letterRunes[rand.IntN(len(letterRunes))])

	return prefix + year + month + day + counter + randomLetter
}
