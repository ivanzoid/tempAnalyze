package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	kIdTemperature  = "T"
	kIdWind         = "Ff"
	kTimeEntryIndex = 0
	kTimeFormat     = "02.01.2006 15:04"
)

type WeatherInfo struct {
	temperature float32
	windSpeed   float32
}

func (w WeatherInfo) String() string {
	return fmt.Sprintf("{T:%.1f, W:%.1f}", w.temperature, w.windSpeed)
}

type AvgWeatherInfo struct {
	yearAvgWeather map[int]*YearAvgWeatherInfo
}

func NewAvgWeatherInfo() *AvgWeatherInfo {
	return &AvgWeatherInfo{yearAvgWeather: make(map[int]*YearAvgWeatherInfo)}
}

func (a *AvgWeatherInfo) String() string {
	years := make([]int, 0, len(a.yearAvgWeather))
	for year, _ := range a.yearAvgWeather {
		years = append(years, year)
	}
	sort.Ints(years)
	var comps []string
	for _, year := range years {
		comps = append(comps, fmt.Sprintf("%v: %v", year, a.yearAvgWeather[year]))
	}
	return fmt.Sprintf("{%v}", strings.Join(comps, "\n"))
}

type YearAvgWeatherInfo struct {
	monthAvgWeather map[int]*MonthAvgWeatherInfo
}

func (y *YearAvgWeatherInfo) String() string {
	months := make([]int, 0, len(y.monthAvgWeather))
	for month, _ := range y.monthAvgWeather {
		months = append(months, month)
	}
	sort.Ints(months)
	var comps []string
	for _, month := range months {
		monthString := time.Month(month).String()[:3]
		comps = append(comps, fmt.Sprintf("%v: %v", monthString, y.monthAvgWeather[month]))
	}
	return fmt.Sprintf("{%v}", strings.Join(comps, "\n"))
}

func NewYearAvgWeatherInfo() *YearAvgWeatherInfo {
	return &YearAvgWeatherInfo{monthAvgWeather: make(map[int]*MonthAvgWeatherInfo)}
}

type MonthAvgWeatherInfo struct {
	hourAvgWeather map[int]*HourAvgWeatherInfo
}

func NewMonthAvgWeatherInfo() *MonthAvgWeatherInfo {
	return &MonthAvgWeatherInfo{hourAvgWeather: make(map[int]*HourAvgWeatherInfo)}
}

func (m *MonthAvgWeatherInfo) String() string {
	hours := make([]int, 0, len(m.hourAvgWeather))
	for hour, _ := range m.hourAvgWeather {
		hours = append(hours, hour)
	}
	sort.Ints(hours)
	var comps []string
	for _, hour := range hours {
		comps = append(comps, fmt.Sprintf("%v: %v", hour, m.hourAvgWeather[hour]))
	}
	return fmt.Sprintf("{%v}", strings.Join(comps, "\n"))
}

type HourAvgWeatherInfo struct {
	temperatureValuesSum      float64
	temperatureValuesQuantity int64
	windValuesSum             float64
	windValuesQuantity        int64
}

func (h *HourAvgWeatherInfo) AvgWeather() WeatherInfo {
	return WeatherInfo{
		temperature: float32(h.temperatureValuesSum / float64(h.temperatureValuesQuantity)),
		windSpeed:   float32(h.windValuesSum / float64(h.windValuesQuantity)),
	}
}

func (h *HourAvgWeatherInfo) String() string {
	avgWeather := h.AvgWeather()
	return fmt.Sprintf("%v", avgWeather)
}

func appName() string {
	return path.Base(os.Args[0])
}

func usage() {
	appName := appName()
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Usage:\n\n")
	fmt.Fprintf(os.Stderr, "\t%v <csvFile1> [<csvFile2> ...]\n", appName)
	fmt.Fprintf(os.Stderr, "\n")
}

func main() {

	flag.Parse()

	if flag.NArg() < 1 {
		usage()
		return
	}

	filePaths := flag.Args()

	weatherInfos := make(map[int64]WeatherInfo)

	for _, filePath := range filePaths {

		file, err := os.Open(filePath)

		if err != nil {
			fmt.Printf("%v: %v\n", filePath, err)
			continue
		}

		defer file.Close()

		hashRune, _ := utf8.DecodeRuneInString("#")
		semicolonRune, _ := utf8.DecodeRuneInString(";")

		reader := csv.NewReader(file)
		reader.Comma = semicolonRune
		reader.Comment = hashRune
		reader.FieldsPerRecord = -1
		lines, err := reader.ReadAll()

		if err != nil {
			fmt.Printf("%v: %v\n", filePath, err)
			continue
		}

		if len(lines) == 0 {
			continue
		}

		infoLine := lines[0]

		temperatureEntryIndex := -1
		windEntryIndex := -1

		for index, entry := range infoLine {
			if entry == kIdTemperature {
				temperatureEntryIndex = index
			} else if entry == kIdWind {
				windEntryIndex = index
			}
		}

		for index, line := range lines {

			if index == 0 {
				continue
			}

			dateTimeString := line[kTimeEntryIndex]
			time, err := time.Parse(kTimeFormat, dateTimeString)
			if err != nil {
				fmt.Printf("%v: can't parse date at line %d\n", filePath, index)
				continue
			}
			dateTime := time.Unix()

			if temperatureEntryIndex >= len(line) {
				fmt.Printf("%v: ill-formed line %d\n", filePath, index)
				continue
			}

			temperatureString := line[temperatureEntryIndex]
			temperature, err := strconv.ParseFloat(temperatureString, 32)
			if err != nil {
				fmt.Printf("%v: can't parse temperature at line %d\n", filePath, index)
				continue
			}

			if windEntryIndex >= len(line) {
				fmt.Printf("%v: ill-formed line %d\n", filePath, index)
				continue
			}

			windString := line[windEntryIndex]
			wind, err := strconv.ParseFloat(windString, 32)
			if err != nil {
				fmt.Printf("%v: can't parse wind at line %d\n", filePath, index)
				continue
			}

			var weatherInfo WeatherInfo

			weatherInfo.temperature = float32(temperature)
			weatherInfo.windSpeed = float32(wind)

			weatherInfos[dateTime] = weatherInfo
		}
	}

	fmt.Println(weatherInfos)

	avgWeather := NewAvgWeatherInfo()

	for dateTime, weatherInfo := range weatherInfos {
		time := time.Unix(dateTime, 0).UTC()

		var yearInfo *YearAvgWeatherInfo
		yearInfo, ok := avgWeather.yearAvgWeather[time.Year()]
		if !ok {
			yearInfo = NewYearAvgWeatherInfo()
			avgWeather.yearAvgWeather[time.Year()] = yearInfo
		}

		var monthInfo *MonthAvgWeatherInfo
		monthInfo, ok = yearInfo.monthAvgWeather[int(time.Month())]
		if !ok {
			monthInfo = NewMonthAvgWeatherInfo()
			yearInfo.monthAvgWeather[int(time.Month())] = monthInfo
		}

		var hourInfo *HourAvgWeatherInfo
		hourInfo, ok = monthInfo.hourAvgWeather[time.Hour()]
		if !ok {
			hourInfo = new(HourAvgWeatherInfo)
			monthInfo.hourAvgWeather[time.Hour()] = hourInfo
		}

		hourInfo.temperatureValuesSum += float64(weatherInfo.temperature)
		hourInfo.temperatureValuesQuantity++

		hourInfo.windValuesSum += float64(weatherInfo.windSpeed)
		hourInfo.windValuesQuantity++
	}

	fmt.Println(avgWeather)
}
