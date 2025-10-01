package main

import (
	"context"
	"log"
	"slices"
	"time"

	"github.com/go-co-op/gocron/v2"

	"git.local.rohrmann.online/arvid/jadehsstundenplansync/v2/internal"
)

type Config struct {
	CalendarUrl     string
	Degree          string
	MaxKWOffset     int
	MinSemester     int
	MaxSemester     int
	ModuleWhitelist []string
}

const (
	DEFAULT_TIMEZONE     = "Europe/Berlin"
	LOG_ERR_SETUP_CONFIG = "Could not setup config, see: %s\n"
	LOG_ERR_SETUP_CALDAV = "Could not setup caldav connection, see: %s\n"
)

func main() {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		panic(err)
	}

	_, err = scheduler.NewJob(gocron.CronJob("0 */4 * * *", false), gocron.NewTask(sync))
	if err != nil {
		panic(err)
	}

	scheduler.Start()

	channel := make(chan bool)

	if res := <-channel; !res {
		err = scheduler.Shutdown()
		if err != nil {
			panic(err)
		}
	}
}

func sync() {
	config, err := internal.SetupConfig()
	if err != nil {
		log.Printf(LOG_ERR_SETUP_CONFIG, err.Error())
		return
	}

	caldavConf := internal.GetCaldavConfiguration(config)

	background := context.Background()

	ctx, timeout := context.WithTimeout(background, 10*60*time.Second)
	defer timeout()

	client, err := internal.SetupCalDav(ctx, caldavConf)
	if err != nil {
		log.Printf(LOG_ERR_SETUP_CALDAV, err.Error())
		panic(err)
	}

	knownEvents, err := internal.GetKnownEvents(client, ctx, caldavConf)
	if err != nil {
		panic(err)
	}

	germanyZone, err := time.LoadLocation(DEFAULT_TIMEZONE)
	if err != nil {
		panic(err)
	}

	timeTable := make([]internal.TimeTableEvent, 0)
	for i := config.MinSemester; i <= config.MaxSemester; i++ {
		for kwOffset := 0; kwOffset <= config.MaxKWOffset; kwOffset++ {
			partialTimeTable, err := internal.GetTimeTable(internal.CreateTimeTableConfig(config.Degree, i, kwOffset))
			if err != nil {
				panic(err)
			}

			partialTimeTable = internal.FilterTimeTableWithWhitelist(config.ModuleWhitelist, partialTimeTable)
			timeTable = append(timeTable, partialTimeTable...)
		}
	}

	eventsInTimeTable := make([]string, 0)
	for _, timeTableEvent := range timeTable {

		id := timeTableEvent.GetEventID()
		eventsInTimeTable = append(eventsInTimeTable, id)
		if !slices.Contains(knownEvents, id) {

			req := internal.AddEventRequest{
				CalendarUrl: config.CalendarUrl,
				Id:          id,
				Event:       timeTableEvent,
				TimeZone:    *germanyZone,
			}

			err = internal.AddEvent(client, ctx, caldavConf, req)
			if err != nil {
				panic(err)
			}

		}

	}

	for _, event := range knownEvents {
		if !slices.Contains(eventsInTimeTable, event) {
			err = internal.DeleteEvent(client, ctx, caldavConf, event)
			if err != nil {
				panic(err)
			}
		}
	}
}
