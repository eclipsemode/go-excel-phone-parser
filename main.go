package main

import (
	"flag"
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
	City     = "Москва"
	FromDate = 2021
	ToDate   = 2024
)

// FlagOptions define the options for flags.
//
// Path specifies the path for xlsx files folder.
//
// City specifies a city looking for.
//
// From describes the year the search starts from.
//
// To describes the last year in which the search ends.
type FlagOptions struct {
	Path string
	City string
	From int
	To   int
}

func main() {
	fmt.Println("Parsing flags...")
	options := readFlags()

	entries, err := os.ReadDir(options.Path)
	if err != nil {
		log.Fatal(err)
	}

	var allPhones []string
	var muAllPhones sync.Mutex
	var wgAllPhones sync.WaitGroup
	wgAllPhones.Add(len(entries))

	fmt.Println("Reading spreadsheets...")
	for _, entry := range entries {
		go appendAllNumbers(&wgAllPhones, &muAllPhones, entry, &allPhones, options)
	}

	wgAllPhones.Wait()
	fmt.Println("Removing repetitions...")
	err = removeRepetitions(&allPhones)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Writing result xlsx file...")
	err = writeResultPhones(&allPhones)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Done: %d phones parsed.\n", len(allPhones))
}

func writeResultPhones(phones *[]string) error {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	index, err := f.NewSheet("Sheet1")
	if err != nil {
		return err
	}

	f.SetActiveSheet(index)

	for i, phone := range *phones {
		err := f.SetCellValue("Sheet1", fmt.Sprintf("A%d", i+1), phone)
		if err != nil {
			return err
		}
	}

	if err := f.SaveAs("./result/base.xlsx"); err != nil {
		return err
	}

	return nil
}

// readFlags read flags (path, city, from and to).
func readFlags() FlagOptions {
	dataPath := flag.String("path", DataPath, "Path for xlsx files")
	city := flag.String("city", City, "City")
	fromDate := flag.Int("from", FromDate, "From date")
	toDate := flag.Int("to", ToDate, "To date")
	flag.Parse()

	return FlagOptions{*dataPath, *city, *fromDate, *toDate}
}

// readSpreadsheets reading spreadsheets in xlsx format
func readSpreadsheets(fileName string, options FlagOptions) []string {
	f, err := excelize.OpenFile(fmt.Sprintf("%s/%s", options.Path, fileName))
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
		go appendColumnNumbers(cities[i], dates[i], phones[i], &result, &wg, &mu, options)
	}

	wg.Wait()
	return result
}

// appendAllNumbers append to array string all numbers from all entries in directory.
func appendAllNumbers(wg *sync.WaitGroup, mu *sync.Mutex, entry os.DirEntry, allPhones *[]string, options FlagOptions) {
	defer wg.Done()
	if !entry.IsDir() {
		phones := readSpreadsheets(entry.Name(), options)

		mu.Lock()
		*allPhones = append(*allPhones, phones...)
		mu.Unlock()
	}
}

// appendColumnNumbers append all number in xlsx file column.
func appendColumnNumbers(
	city string,
	date string,
	phone string,
	res *[]string,
	wg *sync.WaitGroup,
	mutex *sync.Mutex,
	options FlagOptions,
) {
	defer wg.Done()
	if city != options.City {
		return
	}

	year, err := getLastOrderYear(date)
	if err != nil {
		return
	}

	if year < options.From || year > options.To {
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

// parsePhone formatting phone number to format +X(XXX)XXX-XX-XX.
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

// getLastOrderYear returning the last year when the order was carried out.
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

// removeRepetitions removes similar phone numbers from string of array.
func removeRepetitions(phones *[]string) error {
	uniquePhones := make(map[string]struct{})
	chPhone := make(chan string)

	var mu sync.Mutex

	var result []string

	go func() {
		defer close(chPhone)

		for _, phone := range *phones {
			mu.Lock()
			if _, ok := uniquePhones[phone]; !ok {
				uniquePhones[phone] = struct{}{}
				chPhone <- phone
			}
			mu.Unlock()
		}
	}()

	for {
		phone, opened := <-chPhone
		if !opened {
			break
		}

		result = append(result, phone)
	}

	*phones = result

	return nil
}
