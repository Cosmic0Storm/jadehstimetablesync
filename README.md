# Jade HS TimeTable Sync

This litte programm allows to sync the timetable information found under https://www.jade-hs.de/veranstaltungsplaene/ and sync the time table to a caldav server.

The easiest start is via a installed docker instance and working docker compose.
Set your caldav credentials in the .secrets.env file:
```.env
CALENDAR_USERNAME=<username>
CALENDAR_PASSWORD=<password / application token>
```

and change the config.yml
```yml
---
CalendarUrl: <calendar url>
Degree: "MIT Winf" # your degree
MaxKWOffset: 3 # Max Weeks to look into the future
MinSemester: 1 # the lowest semester to search for modules
MaxSemester: 4 # the highest semester to search for modules 
Timezone: Europe/Berlin 
CronSchedule: "0 0,8,14,22 * * *" # tells the programm to sync the time table info at midnight, 8am, 2pm and 10pm, see cron notation

ModuleWhitelist: # whitelist of modules by name, copy paste the names from the time table 
  - "Wissenschaftliches Arbeiten"
  - "Datenbanken"
  - "Betriebliche Anwendungssysteme"
  - "IT-Controlling"
  - "Datenkommunikation"
  - "Marketing und Strategie"
```
and use
```bash
docker compose up -d
```

## Without Docker
Requirements:
1. go >= 1.24.7
2. config.yml seen above in current path
```bash
CALENDAR_USERNAME=<username> CALENDAR_PASSWORD=<password / application token> go run main.go
```
