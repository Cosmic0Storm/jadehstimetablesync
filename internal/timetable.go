package internal

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	iconv "github.com/djimenez/iconv-go"
	"github.com/emersion/go-ical"
)

type TimeTableConfig struct {
	Degree    string
	Semester  int
	Timestamp time.Time
	KWOffset  int
	Timezone  time.Location
}

func (config *TimeTableConfig) getIdentifier() string {
	return config.Degree + strconv.Itoa(config.Semester)
}

func (config *TimeTableConfig) getKW() int {
	_, week := config.Timestamp.ISOWeek()

	return week + config.KWOffset
}

func CreateTimeTableConfig(degree string, semester, kwoffset int) *TimeTableConfig {
	return &TimeTableConfig{Degree: degree, Semester: semester, KWOffset: kwoffset, Timestamp: time.Now()}
}

type TimeTableEvent struct {
	Begin    time.Time
	End      time.Time
	Lecturer string
	Room     string
	Name     string
	Degree   string
	Semester int
	TimeZone *time.Location
}

func (t *TimeTableEvent) GetEventID() string {
	return fmt.Sprintf("%d_%s_%s_%s.ics",
		t.Semester,
		t.Degree,
		t.Room,
		t.Begin.Format(time.RFC3339),
	)
}

type TimeTableColumn struct {
	Weekday     string
	BeginColumn int
	EndColumn   int
}

func CreateSimpleTimeTableColumn(column int) *TimeTableColumn {
	return &TimeTableColumn{BeginColumn: column, EndColumn: column}
}

func (t *TimeTableEvent) GetICalRepr(zone *time.Location) *ical.Calendar {

	event := ical.NewEvent()
	event.Name = ical.CompEvent
	event.DateTimeStart(t.Begin.Location())
	event.DateTimeEnd(t.End.Location())
	event.Props.SetText(ical.PropProductID, "jadecal")
	event.Props.SetDateTime(ical.PropDateTimeStart, t.Begin)
	event.Props.SetDateTime(ical.PropDateTimeEnd, t.End)
	event.Props.SetText(ical.PropContact, t.Lecturer)
	event.Props.SetText(ical.PropLocation, t.Room)
	event.Props.SetText(ical.PropUID, t.GetEventID())
	event.Props.SetText(ical.PropSummary, t.Name)
	event.Props.SetDateTime(ical.PropDateTimeStamp, time.Now())

	calendar := ical.NewCalendar()
	calendar.Props.SetText(ical.PropProductID, "-//jade-hs//SplanCalDavSync//EN")
	calendar.Props.SetText(ical.PropVersion, "2.0")
	calendar.Children = append(calendar.Children, event.Component)

	return calendar
}

func GetTimeTable(config *TimeTableConfig) ([]TimeTableEvent, error) {
	items := make([]TimeTableEvent, 0)

	params := url.Values{}
	params.Add("Bezeichnung", config.getIdentifier())
	params.Add("path", `Studenten-Sets`)
	params.Add("template", `Set3`)
	params.Add("KW", strconv.Itoa(config.getKW()))
	params.Add("days", `1-6`)
	body := strings.NewReader(params.Encode())

	req, err := http.NewRequest(http.MethodPost, "https://www.jade-hs.de/apps/infosys/splan.php", body)
	if err != nil {
		return items, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return items, err
	}
	defer resp.Body.Close()

	utfBody, err := iconv.NewReader(resp.Body, "ISO-8859-1", "UTF-8")
	if err != nil {
		return items, err
	}

	document, err := goquery.NewDocumentFromReader(utfBody)
	if err != nil {
		return items, err
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

	columns := [...]TimeTableColumn{
		TimeTableColumn{Weekday: "Montag", BeginColumn: 1, EndColumn: 1},
		TimeTableColumn{Weekday: "Dienstag", BeginColumn: 2, EndColumn: 2},
		TimeTableColumn{Weekday: "Mittwoch", BeginColumn: 3, EndColumn: 3},
		TimeTableColumn{Weekday: "Donnerstag", BeginColumn: 4, EndColumn: 4},
		TimeTableColumn{Weekday: "Freitag", BeginColumn: 5, EndColumn: 5},
		TimeTableColumn{Weekday: "Samstag", BeginColumn: 5, EndColumn: 5},
	}

	for rowIndex, row := range rows.EachIter() {
		if rowIndex == 0 {
			columnHeaders := row.Find(".col-label-one")
			for _, header := range columnHeaders.EachIter() {
				val, exists := header.Attr("colspan")
				if !exists {
					panic(exists)
				}
				colspan, err := strconv.Atoi(val)
				if err != nil {
					panic(err)
				}

				if colspan == 1 {
					continue
				}

				columns[header.Index()-1].EndColumn = columns[header.Index()-1].BeginColumn + colspan - 1
				for i := header.Index(); i < len(columns); i++ {
					columns[i].BeginColumn = columns[i-1].EndColumn + 1
					columns[i].EndColumn = columns[i].BeginColumn
				}
			}
		}

		events := row.Children()
		firstCellText := row.Find(".row-label-one").First().Text()
		if firstCellText == "" {
			continue
		}
		startTime, err := time.ParseInLocation("15:04", row.Find(".row-label-one").First().Text(), &config.Timezone)
		if err != nil {
			return items, err
		}
		for column, event := range events.EachIter() {
			if !event.Is(".object-cell-border") {
				continue
			}

			weekday := 0

			for weekdayOffset, col := range columns {
				if col.BeginColumn == col.EndColumn && column == col.BeginColumn {
					weekday = weekdayOffset
				}
				if column >= col.BeginColumn && column <= col.EndColumn {
					weekday = weekdayOffset
				}
			}

			daysToAddToStartEvent := daysToAddToStartEventWithoutWeekday + weekday - 1

			startTime := startTime.AddDate(config.Timestamp.Year(), 0, daysToAddToStartEvent)
			rowspan, exists := event.Attr("rowspan")
			if !exists {
				return items, err
			}
			rowSpan, err := strconv.Atoi(rowspan)
			if err != nil {
				return items, err
			}
			endTime := startTime.Add(time.Duration(rowSpan*15) * time.Minute)

			item := TimeTableEvent{Begin: startTime, End: endTime, Semester: config.Semester, Degree: config.Degree}

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

	return items, nil
}

func FilterTimeTableWithWhitelist(whitelist []string, events []TimeTableEvent) []TimeTableEvent {
	if len(whitelist) == 0 {
		return events
	}

	filtered := make([]TimeTableEvent, 0)

	for _, event := range events {

		if slices.Contains(whitelist, event.Name) {
			filtered = append(filtered, event)
		}
	}

	return filtered
}
