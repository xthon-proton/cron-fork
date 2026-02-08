package cronmini

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// ParseOption configures which fields the parser expects in a cron expression.
type ParseOption int

const (
	Second         ParseOption = 1 << iota // Seconds field (0-59)
	SecondOptional                         // Seconds field is optional
	Minute                                 // Minutes field (0-59)
	Hour                                   // Hours field (0-23)
	Dom                                    // Day of month field (1-31)
	Month                                  // Month field (1-12)
	Dow                                    // Day of week field (0-6)
	DowOptional                            // Day of week is optional
	Descriptor                             // Enable descriptors (@hourly, @daily, etc.)
)

var places = []ParseOption{Second, Minute, Hour, Dom, Month, Dow}
var defaults = []string{"0", "0", "0", "*", "*", "*"}

// Parser parses cron expressions with configurable field options.
type Parser struct {
	options ParseOption
}

// ParserOption configures a Parser.
type ParserOption func(*Parser)

// WithSeconds configures the parser to expect a seconds field (6-field cron).
func WithSeconds() ParserOption {
	return func(p *Parser) {
		p.options = Second | Minute | Hour | Dom | Month | Dow | Descriptor
	}
}

// WithOptions sets custom parse options.
func WithOptions(opts ParseOption) ParserOption {
	return func(p *Parser) {
		p.options = opts
	}
}

// NewParser creates a Parser with the given options.
// By default, it uses standard 5-field cron format with descriptors.
func NewParser(opts ...ParserOption) *Parser {
	p := &Parser{
		options: Minute | Hour | Dom | Month | Dow | Descriptor,
	}
	for _, opt := range opts {
		opt(p)
	}

	// Validate: only one optional field allowed
	optionals := 0
	if p.options&DowOptional > 0 {
		optionals++
	}
	if p.options&SecondOptional > 0 {
		optionals++
	}
	if optionals > 1 {
		panic("only one optional field may be configured")
	}

	return p
}

// Parse parses a cron expression and returns a Schedule.
// Supports:
//   - Standard 5-field cron: "minute hour dom month dow"
//   - Extended 6-field cron (with seconds): "second minute hour dom month dow"
//   - Descriptors: @yearly, @annually, @monthly, @weekly, @daily, @midnight, @hourly
//   - Intervals: @every <duration> (e.g., @every 5m, @every 1h30m)
//   - Timezone: CRON_TZ=<timezone> or TZ=<timezone> prefix
func (p *Parser) Parse(spec string) (Schedule, error) {
	if len(spec) == 0 {
		return nil, fmt.Errorf("empty spec string")
	}

	// Extract timezone if present
	loc := time.Local
	if strings.HasPrefix(spec, "TZ=") || strings.HasPrefix(spec, "CRON_TZ=") {
		i := strings.Index(spec, " ")
		eq := strings.Index(spec, "=")
		if i < 0 {
			return nil, fmt.Errorf("missing space after timezone")
		}
		var err error
		loc, err = time.LoadLocation(spec[eq+1 : i])
		if err != nil {
			return nil, fmt.Errorf("invalid timezone %s: %v", spec[eq+1:i], err)
		}
		spec = strings.TrimSpace(spec[i:])
	}

	// Handle descriptors
	if strings.HasPrefix(spec, "@") {
		if p.options&Descriptor == 0 {
			return nil, fmt.Errorf("descriptors not enabled: %s", spec)
		}
		return parseDescriptor(spec, loc)
	}

	// Parse standard/extended cron expression
	fields := strings.Fields(spec)
	fields, err := normalizeFields(fields, p.options)
	if err != nil {
		return nil, err
	}

	return p.parseFields(fields, loc)
}

func (p *Parser) parseFields(fields []string, loc *time.Location) (*SpecSchedule, error) {
	var err error
	getFieldBits := func(field string, b bounds) uint64 {
		if err != nil {
			return 0
		}
		var bits uint64
		bits, err = getField(field, b)
		return bits
	}

	schedule := &SpecSchedule{
		Second:   getFieldBits(fields[0], seconds),
		Minute:   getFieldBits(fields[1], minutes),
		Hour:     getFieldBits(fields[2], hours),
		Dom:      getFieldBits(fields[3], dom),
		Month:    getFieldBits(fields[4], months),
		Dow:      getFieldBits(fields[5], dow),
		Location: loc,
	}

	if err != nil {
		return nil, err
	}

	return schedule, nil
}

// normalizeFields validates and fills in default values for missing fields.
func normalizeFields(fields []string, options ParseOption) ([]string, error) {
	optionals := 0
	if options&SecondOptional > 0 {
		options |= Second
		optionals++
	}
	if options&DowOptional > 0 {
		options |= Dow
		optionals++
	}
	if optionals > 1 {
		return nil, fmt.Errorf("multiple optionals not allowed")
	}

	// Count expected fields
	max := 0
	for _, place := range places {
		if options&place > 0 {
			max++
		}
	}
	min := max - optionals

	// Validate field count
	if count := len(fields); count < min || count > max {
		if min == max {
			return nil, fmt.Errorf("expected %d fields, found %d: %v", min, count, fields)
		}
		return nil, fmt.Errorf("expected %d-%d fields, found %d: %v", min, max, count, fields)
	}

	// Fill in optional field if not provided
	if min < max && len(fields) == min {
		switch {
		case options&DowOptional > 0:
			fields = append(fields, defaults[5])
		case options&SecondOptional > 0:
			fields = append([]string{defaults[0]}, fields...)
		default:
			return nil, fmt.Errorf("unknown optional field")
		}
	}

	// Expand fields to full 6-field format with defaults
	result := make([]string, len(places))
	copy(result, defaults)
	n := 0
	for i, place := range places {
		if options&place > 0 {
			result[i] = fields[n]
			n++
		}
	}

	return result, nil
}

// getField parses a comma-separated list of ranges.
func getField(field string, b bounds) (uint64, error) {
	var bits uint64
	for _, expr := range strings.Split(field, ",") {
		bit, err := getRange(expr, b)
		if err != nil {
			return 0, err
		}
		bits |= bit
	}
	return bits, nil
}

// getRange parses expressions like "5", "1-5", "*/10", "1-10/2".
func getRange(expr string, b bounds) (uint64, error) {
	rangeAndStep := strings.Split(expr, "/")
	lowAndHigh := strings.Split(rangeAndStep[0], "-")
	singleDigit := len(lowAndHigh) == 1

	var start, end, step uint
	var extra uint64
	var err error

	// Parse start and end
	if lowAndHigh[0] == "*" || lowAndHigh[0] == "?" {
		start, end = b.min, b.max
		extra = starBit
	} else {
		start, err = parseIntOrName(lowAndHigh[0], b.names)
		if err != nil {
			return 0, err
		}
		switch len(lowAndHigh) {
		case 1:
			end = start
		case 2:
			end, err = parseIntOrName(lowAndHigh[1], b.names)
			if err != nil {
				return 0, err
			}
		default:
			return 0, fmt.Errorf("too many hyphens: %s", expr)
		}
	}

	// Parse step
	switch len(rangeAndStep) {
	case 1:
		step = 1
	case 2:
		step, err = parseUint(rangeAndStep[1])
		if err != nil {
			return 0, err
		}
		if singleDigit {
			end = b.max
		}
		if step > 1 {
			extra = 0
		}
	default:
		return 0, fmt.Errorf("too many slashes: %s", expr)
	}

	// Validate range
	if start < b.min {
		return 0, fmt.Errorf("value %d below minimum %d: %s", start, b.min, expr)
	}
	if end > b.max {
		return 0, fmt.Errorf("value %d above maximum %d: %s", end, b.max, expr)
	}
	if start > end {
		return 0, fmt.Errorf("start %d beyond end %d: %s", start, end, expr)
	}
	if step == 0 {
		return 0, fmt.Errorf("step must be positive: %s", expr)
	}

	return getBits(start, end, step) | extra, nil
}

// parseIntOrName parses an integer or named value (like "mon" for Monday).
func parseIntOrName(expr string, names map[string]uint) (uint, error) {
	if names != nil {
		if val, ok := names[strings.ToLower(expr)]; ok {
			return val, nil
		}
	}
	return parseUint(expr)
}

// parseUint parses a non-negative integer.
func parseUint(expr string) (uint, error) {
	num, err := strconv.Atoi(expr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse int from %s: %v", expr, err)
	}
	if num < 0 {
		return 0, fmt.Errorf("negative number not allowed: %s", expr)
	}
	return uint(num), nil
}

// getBits returns a bitmask with bits set from min to max, with the given step.
func getBits(min, max, step uint) uint64 {
	if step == 1 {
		return ^(math.MaxUint64 << (max + 1)) & (math.MaxUint64 << min)
	}
	var bits uint64
	for i := min; i <= max; i += step {
		bits |= 1 << i
	}
	return bits
}

// all returns all bits within bounds, plus the star bit.
func all(b bounds) uint64 {
	return getBits(b.min, b.max, 1) | starBit
}

// parseDescriptor handles @yearly, @monthly, @weekly, @daily, @hourly, @every.
func parseDescriptor(desc string, loc *time.Location) (Schedule, error) {
	switch desc {
	case "@yearly", "@annually":
		return &SpecSchedule{
			Second: 1 << seconds.min, Minute: 1 << minutes.min, Hour: 1 << hours.min,
			Dom: 1 << dom.min, Month: 1 << months.min, Dow: all(dow), Location: loc,
		}, nil
	case "@monthly":
		return &SpecSchedule{
			Second: 1 << seconds.min, Minute: 1 << minutes.min, Hour: 1 << hours.min,
			Dom: 1 << dom.min, Month: all(months), Dow: all(dow), Location: loc,
		}, nil
	case "@weekly":
		return &SpecSchedule{
			Second: 1 << seconds.min, Minute: 1 << minutes.min, Hour: 1 << hours.min,
			Dom: all(dom), Month: all(months), Dow: 1 << dow.min, Location: loc,
		}, nil
	case "@daily", "@midnight":
		return &SpecSchedule{
			Second: 1 << seconds.min, Minute: 1 << minutes.min, Hour: 1 << hours.min,
			Dom: all(dom), Month: all(months), Dow: all(dow), Location: loc,
		}, nil
	case "@hourly":
		return &SpecSchedule{
			Second: 1 << seconds.min, Minute: 1 << minutes.min, Hour: all(hours),
			Dom: all(dom), Month: all(months), Dow: all(dow), Location: loc,
		}, nil
	}

	const everyPrefix = "@every "
	if strings.HasPrefix(desc, everyPrefix) {
		duration, err := time.ParseDuration(desc[len(everyPrefix):])
		if err != nil {
			return nil, fmt.Errorf("failed to parse duration %s: %v", desc, err)
		}
		return Every(duration), nil
	}

	return nil, fmt.Errorf("unrecognized descriptor: %s", desc)
}

// Parse is a convenience function that parses a standard 5-field cron expression.
// For extended options, use NewParser().Parse().
func Parse(spec string) (Schedule, error) {
	return NewParser().Parse(spec)
}
