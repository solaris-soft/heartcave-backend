package service

import (
	"testing"
	"time"

	"github.com/solaris-soft/heartcave-backend/internal/db"
)

func TestCalculateUsesServiceDurationAndSkipsOverlaps(t *testing.T) {
	loc := time.FixedZone("test", 0)
	date := time.Date(2026, 5, 14, 0, 0, 0, 0, loc)

	slots := Calculate(date, []db.AdminSchedule{
		{StartTime: "09:00", EndTime: "12:00", SlotMinutes: 60},
	}, []db.Booking{
		{StartTime: "2026-05-14T10:30:00", EndTime: "2026-05-14T11:30:00"},
	}, 90)

	want := []TimeSlot{
		{Start: "2026-05-14T09:00:00", End: "2026-05-14T10:30:00"},
	}
	if len(slots) != len(want) {
		t.Fatalf("len(slots) = %d, want %d: %#v", len(slots), len(want), slots)
	}
	for i := range want {
		if slots[i] != want[i] {
			t.Fatalf("slots[%d] = %#v, want %#v", i, slots[i], want[i])
		}
	}
}

func TestCalculateReturnsEmptyForInvalidSchedule(t *testing.T) {
	date := time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)

	slots := Calculate(date, []db.AdminSchedule{
		{StartTime: "bad", EndTime: "12:00", SlotMinutes: 60},
	}, nil, 60)

	if len(slots) != 0 {
		t.Fatalf("len(slots) = %d, want 0", len(slots))
	}
}
