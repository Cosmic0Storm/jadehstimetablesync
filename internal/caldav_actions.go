package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
)

type LoggingRoundTripper struct{}

func (t LoggingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {

	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		return resp, err
	}

	return resp, err
}

type CaldavConfiguration struct {
	Username    string
	Password    string
	CalendarURL url.URL
}

func (c *CaldavConfiguration) getBase() string {
	return fmt.Sprintf("%s://%s", c.CalendarURL.Scheme, c.CalendarURL.Host)
}

func (c *CaldavConfiguration) getCalendarEndpoint() string {

	return c.CalendarURL.EscapedPath()
}

type AddEventRequest struct {
	CalendarUrl url.URL
	Id          string
	Event       TimeTableEvent
	TimeZone    time.Location
}

func (a *AddEventRequest) GetUrl() string {
	return a.CalendarUrl.JoinPath(a.Id).EscapedPath()
}

func (a *AddEventRequest) GetICalRepr() *ical.Calendar {
	return a.Event.GetICalRepr(&a.TimeZone)
}

func SetupCalDav(ctx context.Context, config CaldavConfiguration) (*caldav.Client, error) {
	httpClient := &http.Client{
		Transport: LoggingRoundTripper{},
	}

	webdavClient := webdav.HTTPClientWithBasicAuth(httpClient, config.Username, config.Password)
	caldavClient, err := caldav.NewClient(webdavClient, config.getBase())
	if err != nil || caldavClient == nil {
		return nil, err
	}

	return caldavClient, nil

}

func GetKnownEvents(client *caldav.Client, ctx context.Context, config CaldavConfiguration) ([]string, error) {

	queryReq := caldav.CalendarQuery{
		CompRequest: caldav.CalendarCompRequest{},
		CompFilter: caldav.CompFilter{
			Name: "VCALENDAR",
		},
	}

	kownEventIDs := make([]string, 0)

	events, err := client.QueryCalendar(ctx, config.getCalendarEndpoint(), &queryReq)
	if err != nil {
		return kownEventIDs, err
	}

	for _, event := range events {
		id := strings.Replace(event.Path, config.CalendarURL.Path, "", 1)[1:]
		id = strings.ReplaceAll(id, "%20", " ")

		kownEventIDs = append(kownEventIDs, id)
	}

	return kownEventIDs, nil
}

func AddEvent(client *caldav.Client, ctx context.Context, config CaldavConfiguration, req AddEventRequest) error {
	_, err := client.PutCalendarObject(ctx, req.GetUrl(), req.GetICalRepr())
	return err
}

func DeleteEvent(client *caldav.Client, ctx context.Context, config CaldavConfiguration, id string) error {
	httpClient := &http.Client{}

	webdavClient := webdav.HTTPClientWithBasicAuth(httpClient, config.Username, config.Password)

	err := client.RemoveAll(ctx, config.CalendarURL.JoinPath(id).EscapedPath())

	req, err := http.NewRequest(http.MethodDelete, config.CalendarURL.JoinPath(id).String(), nil)
	if err != nil {
		return err
	}

	resp, err := webdavClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode == 404 {
		return nil
	}

	return errors.New("Event not deleted")
}
