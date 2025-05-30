package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	logging "appkiller/Logging" // Correct import for local logging package

	"github.com/robfig/cron/v3"
)

func init() {
	// Ensure the logging package is initialized
	logging.Init()
}

func main() {
	appName := flag.String("app", "", "Name of the app/process to kill")
	schedule := flag.String("cron", "*/10 * * * *", "Cron schedule (default: every 10 minutes)")
	atLaunch := flag.Bool("immediate", true, "execute immediately on launch and then follow the cron schedule")
	flag.Parse()

	if *appName == "" {
		logging.Error("Usage Error: appkiller -app <process_name> [-cron <cron_schedule>]")
		fmt.Scanln()
		os.Exit(1)
	}

	c := cron.New()
	_, err := c.AddFunc(*schedule, func() {
		logging.Info("Killing app: %s", *appName)
		killApp(*appName)
	})
	if err != nil {
		logging.Error("Invalid cron schedule: %v", err)
		os.Exit(1)
	}
	if *atLaunch {
		logging.Info("Executing kill immediately for '%s'", *appName)
		killApp(*appName)
	}

	c.Start()
	logging.Info("Scheduled kill for '%s' at cron: %s", *appName, *schedule)
	fmt.Println("Press Enter to exit...")
	fmt.Scanln()
}

func killApp(name string) {
	cmd := exec.Command("taskkill", "/IM", name, "/F")
	out, err := cmd.CombinedOutput()
	if err != nil {
		logging.Error("Failed to kill %s: %v\nOutput: %s", name, err, string(out))
	} else {
		logging.Info("Killed %s", name)
	}
}
