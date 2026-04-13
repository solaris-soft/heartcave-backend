package service

import (
	"fmt"
	"time"

	"github.com/solaris-soft/heartcave-backend/internal/db"
)

// TimeSlot represents a bookable time window.
type TimeSlot struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// Calculate returns available time slots for date given the admin's weekly schedule
// entries for that day and the bookings that already exist on that date.
//
// date must be in "YYYY-MM-DD" format.
// schedule entries are the admin_schedule rows for the matching day_of_week.
// booked are all bookings whose date field equals date.
func Calculate(date string, schedule []db.AdminSchedule, booked []db.Booking) []TimeSlot {
	// Build a set of already-booked start times for fast lookup.
	bookedStarts := make(map[string]struct{}, len(booked))
	for _, b := range booked {
		// Bookings store date as "YYYY-MM-DD HH:MM" or just "HH:MM" depending on
		// how the frontend submits. We normalise to "HH:MM".
		t := extractTime(b.Date)
		if t != "" {
			bookedStarts[t] = struct{}{}
		}
	}

	var slots []TimeSlot
	for _, s := range schedule {
		start, err := time.Parse("15:04", s.StartTime)
		if err != nil {
			continue
		}
		end, err := time.Parse("15:04", s.EndTime)
		if err != nil {
			continue
		}
		duration := time.Duration(s.SlotMinutes) * time.Minute

		for cursor := start; cursor.Add(duration).Before(end) || cursor.Add(duration).Equal(end); cursor = cursor.Add(duration) {
			slotStart := cursor.Format("15:04")
			slotEnd := cursor.Add(duration).Format("15:04")
			if _, taken := bookedStarts[slotStart]; !taken {
				slots = append(slots, TimeSlot{Start: slotStart, End: slotEnd})
			}
		}
	}

	return slots
}

// extractTime pulls "HH:MM" from either "YYYY-MM-DD HH:MM" or "HH:MM".
func extractTime(dateStr string) string {
	// Try full datetime first.
	if t, err := time.Parse("2006-01-02 15:04", dateStr); err == nil {
		return fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
	}
	// Try time only.
	if t, err := time.Parse("15:04", dateStr); err == nil {
		return fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
	}
	return ""
}
