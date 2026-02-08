# cronmini

A refined, minimal, and extensible cron expression parser extracted from the robfig/cron v3 library.

## Features

- **Standard cron parsing**: 5-field format (minute, hour, day-of-month, month, day-of-week)
- **Extended cron parsing**: 6-field format with seconds
- **Descriptor support**: `@hourly`, `@daily`, `@weekly`, `@monthly`, `@yearly`, `@every <duration>`
- **Timezone support**: `CRON_TZ=` or `TZ=` prefix
- **Named values**: Month names (jan-dec) and weekday names (sun-sat)
- **Range expressions**: `1-5`, `*/10`, `1-10/2`, `5,10,15`
- **Efficient bit-set scheduling**: Fast next-time calculation

## Installation

```go
import "github.com/robfig/cron/v3/cronmini"
```

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "time"
    "github.com/robfig/cron/v3/cronmini"
)

func main() {
    // Parse a standard cron expression (every 5 minutes)
    schedule, err := cronmini.Parse("*/5 * * * *")
    if err != nil {
        panic(err)
    }

    // Get the next scheduled time
    next := schedule.Next(time.Now())
    fmt.Printf("Next run: %v\n", next)

    // Get multiple upcoming times
    for i := 0; i < 5; i++ {
        next = schedule.Next(next)
        fmt.Printf("Then: %v\n", next)
    }
}
```

### Extended Format with Seconds

```go
// Create a parser that expects a seconds field
parser := cronmini.NewParser(cronmini.WithSeconds())

// Parse 6-field cron (every 30 seconds)
schedule, err := parser.Parse("*/30 * * * * *")
```

### Using Descriptors

```go
// Hourly
schedule, _ := cronmini.Parse("@hourly")

// Daily at midnight
schedule, _ := cronmini.Parse("@daily")

// Every 5 minutes
schedule, _ := cronmini.Parse("@every 5m")

// Every 1 hour 30 minutes
schedule, _ := cronmini.Parse("@every 1h30m")
```

### Timezone Support

```go
// Run at 9 AM New York time on weekdays
schedule, _ := cronmini.Parse("CRON_TZ=America/New_York 0 9 * * 1-5")

// Run at midnight UTC
schedule, _ := cronmini.Parse("TZ=UTC 0 0 * * *")
```

## Cron Expression Format

### Standard Format (5 fields)

```
┌───────────── minute (0-59)
│ ┌───────────── hour (0-23)
│ │ ┌───────────── day of month (1-31)
│ │ │ ┌───────────── month (1-12 or jan-dec)
│ │ │ │ ┌───────────── day of week (0-6 or sun-sat, 0=Sunday)
│ │ │ │ │
* * * * *
```

### Extended Format (6 fields)

```
┌───────────── second (0-59)
│ ┌───────────── minute (0-59)
│ │ ┌───────────── hour (0-23)
│ │ │ ┌───────────── day of month (1-31)
│ │ │ │ ┌───────────── month (1-12 or jan-dec)
│ │ │ │ │ ┌───────────── day of week (0-6 or sun-sat)
│ │ │ │ │ │
* * * * * *
```

### Special Characters

| Character | Description | Example |
|-----------|-------------|---------|
| `*` | Any value | `* * * * *` (every minute) |
| `,` | List of values | `0,15,30,45 * * * *` (every 15 minutes) |
| `-` | Range of values | `0-30 * * * *` (first 30 minutes) |
| `/` | Step values | `*/10 * * * *` (every 10 minutes) |
| `?` | Same as `*` | `? * * * *` |

### Descriptors

| Descriptor | Equivalent | Description |
|------------|------------|-------------|
| `@yearly` | `0 0 1 1 *` | January 1st at midnight |
| `@annually` | `0 0 1 1 *` | Same as @yearly |
| `@monthly` | `0 0 1 * *` | First day of month at midnight |
| `@weekly` | `0 0 * * 0` | Sunday at midnight |
| `@daily` | `0 0 * * *` | Every day at midnight |
| `@midnight` | `0 0 * * *` | Same as @daily |
| `@hourly` | `0 * * * *` | Every hour at minute 0 |
| `@every <dur>` | - | Every duration (e.g., `@every 5m`) |

## API Reference

### Types

```go
// Schedule represents a cron schedule
type Schedule interface {
    Next(time.Time) time.Time
}

// SpecSchedule stores parsed cron as efficient bit sets
type SpecSchedule struct {
    Second, Minute, Hour, Dom, Month, Dow uint64
    Location *time.Location
}

// ConstantDelaySchedule for fixed intervals
type ConstantDelaySchedule struct {
    Delay time.Duration
}
```

### Functions

```go
// Parse a standard 5-field cron expression
func Parse(spec string) (Schedule, error)

// Create a custom parser
func NewParser(opts ...ParserOption) *Parser

// Parser options
func WithSeconds() ParserOption       // Enable 6-field format
func WithOptions(ParseOption) ParserOption  // Custom field configuration

// Create interval schedule
func Every(duration time.Duration) ConstantDelaySchedule
```

## Day-of-Week and Day-of-Month Logic

When both day-of-week and day-of-month are specified:
- If either field is `*`, they are ANDed (both must match)
- If both fields have specific values, they are ORed (either can match)

This follows standard cron behavior:
- `0 0 15 * 1` → Run at midnight on the 15th OR on Mondays
- `0 0 15 * *` → Run at midnight on the 15th of any month
- `0 0 * * 1` → Run at midnight on every Monday

## Performance

The package uses bit sets for efficient schedule matching:
- Parsing is O(n) where n is the expression length
- Next time calculation is typically O(1) for common cases

Benchmarks (on typical hardware):
```
BenchmarkParse    ~500ns/op
BenchmarkNext     ~100ns/op
```

## License

This package is derived from [robfig/cron](https://github.com/robfig/cron) and is licensed under the MIT License.
