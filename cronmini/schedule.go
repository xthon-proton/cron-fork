package cronmini

import "time"

// Schedule represents a cron schedule that can compute the next activation time.
type Schedule interface {
	// Next returns the next activation time after the given time.
	// If no valid time can be found (within a reasonable limit), returns zero time.
	Next(time.Time) time.Time
}

// SpecSchedule stores the parsed cron specification as bit sets for efficient matching.
// Each field is represented as a uint64 where each bit corresponds to a valid value.
type SpecSchedule struct {
	Second   uint64         // 0-59: bit N is set if second N is valid
	Minute   uint64         // 0-59: bit N is set if minute N is valid
	Hour     uint64         // 0-23: bit N is set if hour N is valid
	Dom      uint64         // 1-31: bit N is set if day N is valid
	Month    uint64         // 1-12: bit N is set if month N is valid
	Dow      uint64         // 0-6: bit N is set if weekday N is valid (0=Sunday)
	Location *time.Location // Timezone for the schedule
}

// bounds defines the valid range and optional name mappings for a cron field.
type bounds struct {
	min, max uint
	names    map[string]uint
}

// Field bounds for each cron component.
var (
	seconds = bounds{0, 59, nil}
	minutes = bounds{0, 59, nil}
	hours   = bounds{0, 23, nil}
	dom     = bounds{1, 31, nil}
	months  = bounds{1, 12, map[string]uint{
		"jan": 1, "feb": 2, "mar": 3, "apr": 4,
		"may": 5, "jun": 6, "jul": 7, "aug": 8,
		"sep": 9, "oct": 10, "nov": 11, "dec": 12,
	}}
	dow = bounds{0, 6, map[string]uint{
		"sun": 0, "mon": 1, "tue": 2, "wed": 3,
		"thu": 4, "fri": 5, "sat": 6,
	}}
)

const (
	// starBit marks fields that have a wildcard (*).
	// Used to determine if day-of-week and day-of-month should be ORed or ANDed.
	starBit = 1 << 63
)

// Next returns the next time this schedule is activated, greater than the given time.
// If no valid time can be found within 5 years, returns the zero time.
func (s *SpecSchedule) Next(t time.Time) time.Time {
	// Convert to schedule's timezone if specified
	origLocation := t.Location()
	loc := s.Location
	if loc == time.Local {
		loc = t.Location()
	}
	if s.Location != time.Local {
		t = t.In(s.Location)
	}

	// Start at the next second
	t = t.Add(1*time.Second - time.Duration(t.Nanosecond())*time.Nanosecond)

	added := false
	yearLimit := t.Year() + 5

WRAP:
	if t.Year() > yearLimit {
		return time.Time{}
	}

	// Find matching month
	for 1<<uint(t.Month())&s.Month == 0 {
		if !added {
			added = true
			t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, loc)
		}
		t = t.AddDate(0, 1, 0)
		if t.Month() == time.January {
			goto WRAP
		}
	}

	// Find matching day
	for !s.dayMatches(t) {
		if !added {
			added = true
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		}
		t = t.AddDate(0, 0, 1)
		// Handle DST transitions
		if t.Hour() != 0 {
			if t.Hour() > 12 {
				t = t.Add(time.Duration(24-t.Hour()) * time.Hour)
			} else {
				t = t.Add(time.Duration(-t.Hour()) * time.Hour)
			}
		}
		if t.Day() == 1 {
			goto WRAP
		}
	}

	// Find matching hour
	for 1<<uint(t.Hour())&s.Hour == 0 {
		if !added {
			added = true
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, loc)
		}
		t = t.Add(1 * time.Hour)
		if t.Hour() == 0 {
			goto WRAP
		}
	}

	// Find matching minute
	for 1<<uint(t.Minute())&s.Minute == 0 {
		if !added {
			added = true
			t = t.Truncate(time.Minute)
		}
		t = t.Add(1 * time.Minute)
		if t.Minute() == 0 {
			goto WRAP
		}
	}

	// Find matching second
	for 1<<uint(t.Second())&s.Second == 0 {
		if !added {
			added = true
			t = t.Truncate(time.Second)
		}
		t = t.Add(1 * time.Second)
		if t.Second() == 0 {
			goto WRAP
		}
	}

	return t.In(origLocation)
}

// dayMatches checks if the time matches the day-of-week and day-of-month constraints.
func (s *SpecSchedule) dayMatches(t time.Time) bool {
	domMatch := 1<<uint(t.Day())&s.Dom > 0
	dowMatch := 1<<uint(t.Weekday())&s.Dow > 0

	// If either field has a wildcard (*), use AND logic
	if s.Dom&starBit > 0 || s.Dow&starBit > 0 {
		return domMatch && dowMatch
	}
	// Otherwise, use OR logic (standard cron behavior)
	return domMatch || dowMatch
}

// ConstantDelaySchedule represents a fixed interval schedule (e.g., "every 5 minutes").
type ConstantDelaySchedule struct {
	Delay time.Duration
}

// Every creates a schedule that triggers every specified duration.
// Delays less than 1 second are rounded up to 1 second.
func Every(duration time.Duration) ConstantDelaySchedule {
	if duration < time.Second {
		duration = time.Second
	}
	return ConstantDelaySchedule{
		Delay: duration - time.Duration(duration.Nanoseconds())%time.Second,
	}
}

// Next returns the next activation time.
func (s ConstantDelaySchedule) Next(t time.Time) time.Time {
	return t.Add(s.Delay - time.Duration(t.Nanosecond())*time.Nanosecond)
}
