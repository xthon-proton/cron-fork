// Package cronmini provides a refined, minimal, and extensible cron expression parser.
//
// This package extracts the core cron parsing logic from the robfig/cron v3 library,
// offering a clean API for:
//   - Parsing standard cron expressions (5-field: minute, hour, dom, month, dow)
//   - Parsing extended cron expressions (6-field with seconds)
//   - Computing the next scheduled time based on a cron expression
//   - Supporting descriptors like @hourly, @daily, @weekly, @monthly, @yearly, @every
//   - Timezone-aware scheduling with CRON_TZ= prefix
//
// # Basic Usage
//
//	// Parse a standard cron expression
//	schedule, err := cronmini.Parse("*/5 * * * *")  // every 5 minutes
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get the next scheduled time
//	next := schedule.Next(time.Now())
//	fmt.Printf("Next run: %v\n", next)
//
// # Extended Usage with Seconds
//
//	// Create a parser with seconds field
//	parser := cronmini.NewParser(cronmini.WithSeconds())
//	schedule, err := parser.Parse("*/30 * * * * *")  // every 30 seconds
//
// # Descriptors
//
//	schedule, _ := cronmini.Parse("@hourly")     // every hour
//	schedule, _ := cronmini.Parse("@every 5m")  // every 5 minutes
//
// # Timezone Support
//
//	schedule, _ := cronmini.Parse("CRON_TZ=America/New_York 0 9 * * 1-5")  // 9 AM on weekdays, NYC time
package cronmini
