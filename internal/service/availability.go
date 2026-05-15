package service

import (
	"time"

	"github.com/solaris-soft/heartcave-backend/internal/db"
)

const LocalDateTimeLayout = "2006-01-02T15:04:05"

// TimeSlot represents a bookable local time window.
type TimeSlot struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// Calculate returns available slots for a date using weekly schedule entries,
// existing active bookings, and the requested service duration.
func Calculate(date time.Time, schedule []db.AdminSchedule, booked []db.Booking, durationMinutes int64) []TimeSlot {
	if durationMinutes <= 0 {
		durationMinutes = 60
	}

	duration := time.Duration(durationMinutes) * time.Minute
	slots := make([]TimeSlot, 0)

	for _, s := range schedule {
		windowStart, err := parseScheduleTime(date, s.StartTime)
		if err != nil {
			continue
		}
		windowEnd, err := parseScheduleTime(date, s.EndTime)
		if err != nil {
			continue
		}

		for cursor := windowStart; !cursor.Add(duration).After(windowEnd); cursor = cursor.Add(duration) {
			end := cursor.Add(duration)
			if !overlapsAny(cursor, end, booked, date.Location()) {
				slots = append(slots, TimeSlot{
					Start: cursor.Format(LocalDateTimeLayout),
					End:   end.Format(LocalDateTimeLayout),
				})
			}
		}
	}

	return slots
}

func parseScheduleTime(date time.Time, value string) (time.Time, error) {
	clock, err := time.Parse("15:04", value)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(date.Year(), date.Month(), date.Day(), clock.Hour(), clock.Minute(), 0, 0, date.Location()), nil
}

func overlapsAny(start, end time.Time, booked []db.Booking, loc *time.Location) bool {
	for _, booking := range booked {
		bookedStart, err := time.ParseInLocation(LocalDateTimeLayout, booking.StartTime, loc)
		if err != nil {
			continue
		}
		bookedEnd, err := time.ParseInLocation(LocalDateTimeLayout, booking.EndTime, loc)
		if err != nil {
			continue
		}
		if start.Before(bookedEnd) && end.After(bookedStart) {
			return true
		}
	}
	return false
}
