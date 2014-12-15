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

//
// WeatherSample
//

type WeatherSample struct {
	temperature float32
	windSpeed   float32
}

func (w WeatherSample) String() string {
	return fmt.Sprintf("{T:%.1f, W:%.1f}", w.temperature, w.windSpeed)
}

//
// HourAvgWeather
//

type HourAvgWeather struct {
	temperatureValuesSum      float64
	temperatureValuesQuantity int64
	windValuesSum             float64
	windValuesQuantity        int64
}

func (h *HourAvgWeather) AddWeatherData(weather WeatherSample) {
	h.temperatureValuesSum += float64(weather.temperature)
	h.temperatureValuesQuantity++

	h.windValuesSum += float64(weather.windSpeed)
	h.windValuesQuantity++
}

func (h *HourAvgWeather) AvgWeather() WeatherSample {
	return WeatherSample{
		temperature: float32(h.temperatureValuesSum / float64(h.temperatureValuesQuantity)),
		windSpeed:   float32(h.windValuesSum / float64(h.windValuesQuantity)),
	}
}

func (h *HourAvgWeather) String() string {
	avgWeather := h.AvgWeather()
	return fmt.Sprintf("%v", avgWeather)
}

//
// MonthAvgWeather
//

type MonthAvgWeather struct {
	hourAvgWeather map[int]*HourAvgWeather
}

func NewMonthAvgWeather() *MonthAvgWeather {
	return &MonthAvgWeather{hourAvgWeather: make(map[int]*HourAvgWeather)}
}

func (m *MonthAvgWeather) String() string {
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

func (m *MonthAvgWeather) HourAvgWeatherForTime(time time.Time) *HourAvgWeather {
	var hourWeather *HourAvgWeather
	hourWeather, ok := m.hourAvgWeather[time.Hour()]
	if !ok {
		hourWeather = new(HourAvgWeather)
		m.hourAvgWeather[time.Hour()] = hourWeather
	}
	return hourWeather
}

//
// YearAvgWeather
//

type YearAvgWeather struct {
	monthAvgWeather map[int]*MonthAvgWeather
}

func NewYearAvgWeather() *YearAvgWeather {
	return &YearAvgWeather{monthAvgWeather: make(map[int]*MonthAvgWeather)}
}

func (y *YearAvgWeather) String() string {
	months := make([]int, 0, len(y.monthAvgWeather))
	for month, _ := range y.monthAvgWeather {
		months = append(months, month)
	}
	sort.Ints(months)
	var comps []string
	for _, month := range months {
		monthString := time.Month(month).String()[:3]
		comps = append(comps, fmt.Sprintf("%v:\n%v", monthString, y.monthAvgWeather[month]))
	}
	return fmt.Sprintf("{%v}", strings.Join(comps, "\n"))
}

func (y *YearAvgWeather) MonthAvgWeatherForTime(time time.Time) *MonthAvgWeather {
	var monthWeather *MonthAvgWeather
	monthWeather, ok := y.monthAvgWeather[int(time.Month())]
	if !ok {
		monthWeather = NewMonthAvgWeather()
		y.monthAvgWeather[int(time.Month())] = monthWeather
	}
	return monthWeather
}

//
// AvgWeather
//

type AvgWeather struct {
	yearAvgWeather     map[int]*YearAvgWeather
	allYearsAvgWeather *YearAvgWeather
}

func NewAvgWeather() *AvgWeather {
	return &AvgWeather{
		yearAvgWeather:     make(map[int]*YearAvgWeather),
		allYearsAvgWeather: NewYearAvgWeather(),
	}
}

func (a *AvgWeather) String() string {
	years := make([]int, 0, len(a.yearAvgWeather))
	for year, _ := range a.yearAvgWeather {
		years = append(years, year)
	}
	sort.Ints(years)
	var comps []string
	for _, year := range years {
		comps = append(comps, fmt.Sprintf("%v:\n%v\n", year, a.yearAvgWeather[year]))
	}
	var allYearsString string = ""
	if len(years) != 0 {
		allYearsString = fmt.Sprintf("\n%v-%v:\n%v", years[0], years[len(years)-1], a.allYearsAvgWeather)
	}
	return fmt.Sprintf("{%v%v}\n", strings.Join(comps, "\n"), allYearsString)
}

func (a *AvgWeather) YearAvgWeatherForTime(time time.Time) *YearAvgWeather {
	var yearWeather *YearAvgWeather
	yearWeather, ok := a.yearAvgWeather[time.Year()]
	if !ok {
		yearWeather = NewYearAvgWeather()
		a.yearAvgWeather[time.Year()] = yearWeather
	}
	return yearWeather
}

//
// Utils
//

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

func parceCsvFiles(filePaths []string) map[int64]WeatherSample {
	weatherSamples := make(map[int64]WeatherSample)

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
				fmt.Printf("%v: can't parse date at entry #%d\n", filePath, index)
				continue
			}
			dateTime := time.Unix()

			if temperatureEntryIndex >= len(line) {
				fmt.Printf("%v: ill-formed entry #%d\n", filePath, index)
				continue
			}

			temperatureString := line[temperatureEntryIndex]
			temperature, err := strconv.ParseFloat(temperatureString, 32)
			if err != nil {
				fmt.Printf("%v: can't parse temperature at entry #%d\n", filePath, index)
				continue
			}

			if windEntryIndex >= len(line) {
				fmt.Printf("%v: ill-formed entry #%d\n", filePath, index)
				continue
			}

			windString := line[windEntryIndex]
			wind, err := strconv.ParseFloat(windString, 32)
			if err != nil {
				fmt.Printf("%v: can't parse wind at entry #%d\n", filePath, index)
				continue
			}

			var weather WeatherSample

			weather.temperature = float32(temperature)
			weather.windSpeed = float32(wind)

			weatherSamples[dateTime] = weather
		}
	}

	return weatherSamples
}

func main() {

	flag.Parse()

	if flag.NArg() < 1 {
		usage()
		return
	}

	filePaths := flag.Args()

	weatherSamples := parceCsvFiles(filePaths)

	//fmt.Println(weatherSamples)

	avgWeather := NewAvgWeather()

	for dateTime, weather := range weatherSamples {
		time := time.Unix(dateTime, 0).UTC()

		hourInfo := avgWeather.YearAvgWeatherForTime(time).MonthAvgWeatherForTime(time).HourAvgWeatherForTime(time)
		hourInfo.AddWeatherData(weather)

		allYearsHourInfo := avgWeather.allYearsAvgWeather.MonthAvgWeatherForTime(time).HourAvgWeatherForTime(time)
		allYearsHourInfo.AddWeatherData(weather)
	}

	fmt.Println(avgWeather)
}
