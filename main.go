package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/djimenez/iconv-go"
)

const (
	BusinessInformatics = "MIT Winf"
)

type TimeTableConfig struct {
	Course    string
	Semester  int
	Timestamp time.Time
	KWOffset  int
}

func (config *TimeTableConfig) getIdentifier() string {
	return config.Course + strconv.Itoa(config.Semester)
}

func (config *TimeTableConfig) getKW() int {
	_, week := config.Timestamp.ISOWeek()

	return week + config.KWOffset
}

func CreateTimeTableConfig(course string, semester, kwoffset int) *TimeTableConfig {
	return &TimeTableConfig{Course: course, Semester: semester, KWOffset: kwoffset, Timestamp: time.Now()}
}

type TimeTableItem struct {
	Begin    time.Time
	End      time.Time
	Lecturer string
	Room     string
	Name     string
}

func getTimeTable(config *TimeTableConfig) {

	params := url.Values{}
	params.Add("Bezeichnung", config.getIdentifier())
	params.Add("path", `Studenten-Sets`)
	params.Add("template", `Set3`)
	params.Add("KW", strconv.Itoa(config.getKW()))
	params.Add("days", `1-6`)
	params.Add("periods", `1-56`)
	body := strings.NewReader(params.Encode())

	req, err := http.NewRequest(http.MethodPost, "https://www.jade-hs.de/apps/infosys/splan.php", body)
	if err != nil {
		// handle err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("Could not receive time table from infosys")
	}
	defer resp.Body.Close()

	utfBody, err := iconv.NewReader(resp.Body, "ISO-8859-1", "UTF-8")
	if err != nil {
		slog.Debug("Could not convert the response body to utf8")
	}

	document, err := goquery.NewDocumentFromReader(utfBody)
	if err != nil {
		slog.Debug("Could not get the response Body")
	}

	table := document.Find(".grid-border-args")
	rows := table.Children().First().Children()
	dayofWeek := config.Timestamp.Weekday()
	daysToSubtract := int(dayofWeek) - 1
	if daysToSubtract < 0 {
		daysToSubtract += 7
	}

	currentMonday := config.Timestamp.AddDate(0, 0, -daysToSubtract)
	nextMonday := currentMonday.AddDate(0, 0, 7)
	daysToAddToStartEventWithoutWeekday := nextMonday.YearDay() + (config.KWOffset-1)*7

	items := make([]TimeTableItem, 0)

	for rowIndex, row := range rows.EachIter() {
		if rowIndex == 0 {
			continue
		}

		events := row.Children()
		firstCellText := row.Find(".row-label-one").First().Text()
		if firstCellText == "" {
			continue
		}
		startTime, err := time.Parse("15:04", row.Find(".row-label-one").First().Text())
		if err != nil {
			panic(err)
		}
		for column, event := range events.EachIter() {
			if !event.Is(".object-cell-border") {
				continue
			}

			daysToAddToStartEvent := daysToAddToStartEventWithoutWeekday + column - 2

			startTime := startTime.AddDate(config.Timestamp.Year(), 0, daysToAddToStartEvent)
			rowspan, exists := event.Attr("rowspan")
			if !exists {
				panic(err)
			}
			rowSpan, err := strconv.Atoi(rowspan)
			if err != nil {
				panic(err)
			}
			endTime := startTime.Add(time.Duration(rowSpan*15) * time.Minute)

			item := TimeTableItem{Begin: startTime, End: endTime}

			eventTable := event.Find(".object-cell-args")
			eventCells := eventTable.Find("td")
			for i, text := range eventCells.EachIter() {
				switch i {
				case 0:
					item.Name = text.Text()
				case 1:
					item.Lecturer = text.Text()
				case 2:
					item.Room = text.Text()
				}
			}

			items = append(items, item)

		}
	}

	for index, item := range items {
		fmt.Printf("%d >> Begin: %s End: %s Name: %s Lecturer: %s Room: %s\n", index, item.Begin.Format(time.RFC3339), item.End.Format(time.RFC3339), item.Name, item.Lecturer, item.Room)
	}

}

func main() {
	//	getTimeTable(&TimeTableConfig{Course: "MIT Winf", Semester: 1, KWOffset: 1})
	getTimeTable(CreateTimeTableConfig("MIT Winf", 1, 0))
	getTimeTable(CreateTimeTableConfig("MIT Winf", 1, 1))
	getTimeTable(CreateTimeTableConfig("MIT Winf", 1, 2))
}
