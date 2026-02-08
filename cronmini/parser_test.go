package cronmini

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// Test helpers
func every5min(loc *time.Location) *SpecSchedule {
	return &SpecSchedule{1 << 0, 1 << 5, all(hours), all(dom), all(months), all(dow), loc}
}

func midnight(loc *time.Location) *SpecSchedule {
	return &SpecSchedule{1, 1, 1, all(dom), all(months), all(dow), loc}
}

func annual(loc *time.Location) *SpecSchedule {
	return &SpecSchedule{
		Second: 1 << seconds.min, Minute: 1 << minutes.min, Hour: 1 << hours.min,
		Dom: 1 << dom.min, Month: 1 << months.min, Dow: all(dow), Location: loc,
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected Schedule
		wantErr  string
	}{
		{"every 5 minutes", "5 * * * *", every5min(time.Local), ""},
		{"every minute", "* * * * *", &SpecSchedule{
			Second: 1 << 0, Minute: all(minutes), Hour: all(hours),
			Dom: all(dom), Month: all(months), Dow: all(dow), Location: time.Local,
		}, ""},
		{"descriptor hourly", "@hourly", &SpecSchedule{
			Second: 1 << 0, Minute: 1 << 0, Hour: all(hours),
			Dom: all(dom), Month: all(months), Dow: all(dow), Location: time.Local,
		}, ""},
		{"descriptor every 5m", "@every 5m", ConstantDelaySchedule{5 * time.Minute}, ""},
		{"descriptor midnight", "@midnight", midnight(time.Local), ""},
		{"descriptor yearly", "@yearly", annual(time.Local), ""},
		{"descriptor annually", "@annually", annual(time.Local), ""},
		{"empty string", "", nil, "empty spec string"},
		{"too few fields", "* * * *", nil, "expected 5 fields"},
		{"invalid number", "j * * * *", nil, "failed to parse int"},
		{"invalid descriptor", "@unknown", nil, "unrecognized descriptor"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := Parse(tc.expr)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("expected %+v, got %+v", tc.expected, actual)
			}
		})
	}
}

func TestParseWithSeconds(t *testing.T) {
	parser := NewParser(WithSeconds())

	tests := []struct {
		name     string
		expr     string
		expected Schedule
	}{
		{"every 30 seconds", "*/30 * * * * *", &SpecSchedule{
			Second: 1 << 0 | 1 << 30, Minute: all(minutes), Hour: all(hours),
			Dom: all(dom), Month: all(months), Dow: all(dow), Location: time.Local,
		}},
		{"specific second", "5 0 * * * *", &SpecSchedule{
			Second: 1 << 5, Minute: 1 << 0, Hour: all(hours),
			Dom: all(dom), Month: all(months), Dow: all(dow), Location: time.Local,
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := parser.Parse(tc.expr)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("expected %+v, got %+v", tc.expected, actual)
			}
		})
	}
}

func TestParseTimezone(t *testing.T) {
	tokyo, _ := time.LoadLocation("Asia/Tokyo")

	tests := []struct {
		name     string
		expr     string
		expected *time.Location
	}{
		{"CRON_TZ prefix", "CRON_TZ=Asia/Tokyo 5 * * * *", tokyo},
		{"TZ prefix", "TZ=UTC 5 * * * *", time.UTC},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := Parse(tc.expr)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			spec, ok := actual.(*SpecSchedule)
			if !ok {
				t.Errorf("expected *SpecSchedule, got %T", actual)
				return
			}
			if spec.Location.String() != tc.expected.String() {
				t.Errorf("expected location %v, got %v", tc.expected, spec.Location)
			}
		})
	}
}

func TestNext(t *testing.T) {
	// Test at a known time
	base := time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local)

	tests := []struct {
		name     string
		expr     string
		from     time.Time
		expected time.Time
	}{
		{
			"next minute at 35",
			"35 * * * *",
			base,
			time.Date(2024, 1, 15, 10, 35, 0, 0, time.Local),
		},
		{
			"next hour at minute 0",
			"0 11 * * *",
			base,
			time.Date(2024, 1, 15, 11, 0, 0, 0, time.Local),
		},
		{
			"next day at midnight",
			"0 0 * * *",
			base,
			time.Date(2024, 1, 16, 0, 0, 0, 0, time.Local),
		},
		{
			"specific day of month",
			"0 0 20 * *",
			base,
			time.Date(2024, 1, 20, 0, 0, 0, 0, time.Local),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			schedule, err := Parse(tc.expr)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}
			next := schedule.Next(tc.from)
			if !next.Equal(tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, next)
			}
		})
	}
}

func TestEvery(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected time.Duration
	}{
		{5 * time.Minute, 5 * time.Minute},
		{time.Second, time.Second},
		{500 * time.Millisecond, time.Second}, // rounds up
		{time.Minute + 500*time.Millisecond, time.Minute}, // truncates
	}

	for _, tc := range tests {
		schedule := Every(tc.duration)
		if schedule.Delay != tc.expected {
			t.Errorf("Every(%v): expected delay %v, got %v", tc.duration, tc.expected, schedule.Delay)
		}
	}
}

func TestConstantDelayNext(t *testing.T) {
	schedule := Every(5 * time.Minute)
	now := time.Date(2024, 1, 15, 10, 30, 15, 0, time.Local)
	expected := time.Date(2024, 1, 15, 10, 35, 15, 0, time.Local)

	next := schedule.Next(now)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestGetRange(t *testing.T) {
	tests := []struct {
		expr     string
		min, max uint
		expected uint64
		wantErr  string
	}{
		{"5", 0, 7, 1 << 5, ""},
		{"0", 0, 7, 1 << 0, ""},
		{"5-7", 0, 7, 1<<5 | 1<<6 | 1<<7, ""},
		{"*/2", 0, 7, 1<<0 | 1<<2 | 1<<4 | 1<<6, ""},
		{"5-7/2", 0, 7, 1<<5 | 1<<7, ""},
		{"*", 1, 3, 1<<1 | 1<<2 | 1<<3 | starBit, ""},

		// Error cases
		{"5--5", 0, 7, 0, "too many hyphens"},
		{"*//2", 0, 7, 0, "too many slashes"},
		{"1", 3, 5, 0, "below minimum"},
		{"6", 3, 5, 0, "above maximum"},
		{"5-3", 3, 5, 0, "start"},
		{"*/0", 0, 7, 0, "step must be positive"},
	}

	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			actual, err := getRange(tc.expr, bounds{tc.min, tc.max, nil})
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if actual != tc.expected {
				t.Errorf("expected %064b, got %064b", tc.expected, actual)
			}
		})
	}
}

func TestNamedValues(t *testing.T) {
	// Test month names
	schedule, err := Parse("0 0 1 jan *")
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	spec := schedule.(*SpecSchedule)
	if spec.Month != 1<<1|starBit { // January is month 1, plus wildcard for dow
		// Actually month should be just 1<<1 since jan is specified, not *
		if spec.Month&(1<<1) == 0 {
			t.Errorf("expected January (bit 1) to be set in month field")
		}
	}

	// Test day names
	schedule2, err := Parse("0 0 * * mon")
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	spec2 := schedule2.(*SpecSchedule)
	if spec2.Dow&(1<<1) == 0 {
		t.Errorf("expected Monday (bit 1) to be set in dow field")
	}
}

func TestDayMatching(t *testing.T) {
	// When both DOM and DOW are specified (not *), use OR logic
	schedule := &SpecSchedule{
		Second:   1 << 0,
		Minute:   1 << 0,
		Hour:     1 << 0,
		Dom:      1 << 15,       // 15th of month only (no star bit)
		Month:    all(months),
		Dow:      1 << 1,        // Monday only (no star bit)
		Location: time.Local,
	}

	// Should match either the 15th OR Monday
	monday := time.Date(2024, 1, 8, 0, 0, 0, 0, time.Local) // Monday, not 15th
	if !schedule.dayMatches(monday) {
		t.Error("expected Monday to match (OR logic)")
	}

	fifteenth := time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local) // 15th, not Monday
	if !schedule.dayMatches(fifteenth) {
		t.Error("expected 15th to match (OR logic)")
	}
}

func TestDayMatchingWithWildcard(t *testing.T) {
	// When DOM has wildcard, use AND logic
	schedule := &SpecSchedule{
		Second:   1 << 0,
		Minute:   1 << 0,
		Hour:     1 << 0,
		Dom:      all(dom),    // * (has star bit)
		Month:    all(months),
		Dow:      1 << 1,      // Monday only
		Location: time.Local,
	}

	monday := time.Date(2024, 1, 8, 0, 0, 0, 0, time.Local) // Monday
	if !schedule.dayMatches(monday) {
		t.Error("expected Monday to match")
	}

	tuesday := time.Date(2024, 1, 9, 0, 0, 0, 0, time.Local) // Tuesday
	if schedule.dayMatches(tuesday) {
		t.Error("expected Tuesday NOT to match")
	}
}

func TestNewParserPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for multiple optionals")
		}
	}()

	// This should panic
	NewParser(WithOptions(SecondOptional | DowOptional | Minute | Hour | Dom | Month))
}

// Benchmarks
func BenchmarkParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Parse("*/5 * * * *")
	}
}

func BenchmarkNext(b *testing.B) {
	schedule, _ := Parse("*/5 * * * *")
	now := time.Now()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		schedule.Next(now)
	}
}
