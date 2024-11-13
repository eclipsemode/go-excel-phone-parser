package main

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

const (
	DataPath = "./data"
	City     = "Армавир"
	FromDate = 2021
	ToDate   = 2024
)

func main() {
	entries, err := os.ReadDir(DataPath)
	if err != nil {
		log.Fatal(err)
	}

	var allPhones []string
	var muAllPhones sync.Mutex
	var wgAllPhones sync.WaitGroup
	wgAllPhones.Add(len(entries))

	for _, entry := range entries {
		go appendAllNumbers(&wgAllPhones, &muAllPhones, entry, &allPhones)
	}

	wgAllPhones.Wait()
	fmt.Println(len(allPhones))
}

func readSpreadsheets(path string, fileName string) []string {
	f, err := excelize.OpenFile(fmt.Sprintf("%s/%s", path, fileName))
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()

	var result []string
	var mu sync.Mutex

	cols, err := f.GetCols("Лист1")
	phones := cols[0]
	cities := cols[1]
	dates := cols[2]

	var wg sync.WaitGroup
	wg.Add(len(phones))

	for i := 0; i < len(phones); i++ {
		go appendColumnNumbers(cities[i], dates[i], phones[i], &result, &wg, &mu)
	}

	wg.Wait()
	return result
}

func appendAllNumbers(wg *sync.WaitGroup, mu *sync.Mutex, entry os.DirEntry, allPhones *[]string) {
	defer wg.Done()
	if !entry.IsDir() {
		phones := readSpreadsheets(DataPath, entry.Name())

		mu.Lock()
		*allPhones = append(*allPhones, phones...)
		mu.Unlock()
	}
}

func appendColumnNumbers(city string, date string, phone string, res *[]string, wg *sync.WaitGroup, mutex *sync.Mutex) {
	defer wg.Done()
	if city != City {
		return
	}

	year, err := getLastOrderYear(date)
	if err != nil {
		return
	}

	if year < FromDate || year > ToDate {
		return
	}
	parsedPhone, err := parsePhone(phone)
	if err != nil {
		return
	}

	mutex.Lock()
	*res = append(*res, parsedPhone)
	mutex.Unlock()
}

func parsePhone(phone string) (string, error) {
	re := regexp.MustCompile("[^0-9]")

	resultPhone := re.ReplaceAllString(phone, "")

	for _, r := range resultPhone {
		if !unicode.IsDigit(r) {
			return "", fmt.Errorf("is not a digit")
		}
	}

	if len(resultPhone) == 10 {
		resultPhone = "8" + resultPhone
	}

	if len(resultPhone) == 11 {
		if resultPhone[0] == '8' {
			resultPhone = "7" + resultPhone[1:]
		}
	}

	if len(resultPhone) != 11 {
		return "", fmt.Errorf("invalid phone length")
	}

	if resultPhone[:2] != "79" {
		return "", fmt.Errorf("invalid phone number")
	}

	resultPhone = fmt.Sprintf("+%s(%s)%s-%s-%s", resultPhone[:1], resultPhone[1:4], resultPhone[4:7], resultPhone[7:9], resultPhone[9:11])

	return resultPhone, nil
}

func getLastOrderYear(date string) (int, error) {
	if date == "" {
		return 0, fmt.Errorf("date is empty")
	}

	dateArr := strings.Split(date, "/")

	if len(dateArr) != 3 {
		return 0, fmt.Errorf("invalid date format")
	}

	intYear, err := strconv.Atoi(dateArr[2])
	if err != nil {
		return 0, err
	}

	return intYear, nil
}
