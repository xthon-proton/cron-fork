// Example demonstrating the cronmini package usage.
package main

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3/cronmini"
)

func main() {
	fmt.Println("=== cronmini Examples ===")
	fmt.Println()

	// Example 1: Standard cron expression
	fmt.Println("1. Standard cron (every 5 minutes):")
	schedule, err := cronmini.Parse("*/5 * * * *")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	printNextTimes(schedule, 3)

	// Example 2: Descriptors
	fmt.Println("\n2. Descriptor (@hourly):")
	schedule, _ = cronmini.Parse("@hourly")
	printNextTimes(schedule, 3)

	// Example 3: Every duration
	fmt.Println("\n3. Every 30 seconds:")
	schedule, _ = cronmini.Parse("@every 30s")
	printNextTimes(schedule, 3)

	// Example 4: Specific time
	fmt.Println("\n4. Every day at 9:30 AM:")
	schedule, _ = cronmini.Parse("30 9 * * *")
	printNextTimes(schedule, 3)

	// Example 5: Weekdays only
	fmt.Println("\n5. Weekdays at 8 AM:")
	schedule, _ = cronmini.Parse("0 8 * * 1-5")
	printNextTimes(schedule, 5)

	// Example 6: With seconds
	fmt.Println("\n6. With seconds (every 15 seconds):")
	parser := cronmini.NewParser(cronmini.WithSeconds())
	schedule, _ = parser.Parse("*/15 * * * * *")
	printNextTimes(schedule, 4)

	// Example 7: Timezone
	fmt.Println("\n7. With timezone (9 AM Tokyo time):")
	schedule, _ = cronmini.Parse("CRON_TZ=Asia/Tokyo 0 9 * * *")
	printNextTimes(schedule, 2)

	// Example 8: Named values
	fmt.Println("\n8. Named values (First Monday of each month at 10 AM):")
	schedule, _ = cronmini.Parse("0 10 1-7 * mon")
	printNextTimes(schedule, 3)
}

func printNextTimes(schedule cronmini.Schedule, count int) {
	now := time.Now()
	t := now
	for i := 0; i < count; i++ {
		t = schedule.Next(t)
		fmt.Printf("   %s\n", t.Format("2006-01-02 15:04:05 MST"))
	}
}
