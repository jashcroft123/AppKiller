package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	appMutex "appkiller/AppMutex"
	logging "appkiller/Logging"

	"github.com/getlantern/systray"
	"github.com/robfig/cron/v3"
)

const applicationName = "AppKiller"

//go:embed icon.ico
var iconData []byte

var (
	appToKill string
	schedule  string
	atLaunch  bool
	silent    bool
)

func hideConsole() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	freeConsole := kernel32.NewProc("FreeConsole")
	freeConsole.Call()
}

func init() {
	logging.Init()

	// Parse CLI args early
	flag.StringVar(&appToKill, "app", "", "Name of the app/process to kill")
	flag.StringVar(&schedule, "cron", "*/10 * * * *", "Cron schedule (default: every 10 minutes)")
	flag.BoolVar(&atLaunch, "immediate", true, "Execute immediately on launch")
	flag.BoolVar(&silent, "silent", true, "Execute Silently on launch (no terminal updates)")
	flag.Parse()

	if silent {
		hideConsole()
		logging.Info("Running in silent mode, console will be hidden.")
	}

	// Check if required args are present
	if appToKill == "" {
		fmt.Println("Usage: appkiller -app <process_name> [-cron <cron_schedule>] [-immediate]")
		os.Exit(1)
	}

	// Attempt to acquire mutex
	_, err := appMutex.CreateMutex(applicationName + "Mutex")
	if err != nil {
		logging.Error("Another instance is already running: %v", err)
		fmt.Println("AppKiller is already running.")
		os.Exit(1)
	}

	logging.Info("AppKiller initialized with app='%s', cron='%s', atLaunch=%v", appToKill, schedule, atLaunch)
}

func main() {
	defer handleCrash()
	systray.Run(onReady, onExit)
}

func dispose() {
	systray.Quit()
}

func handleCrash() {
	if r := recover(); r != nil {
		logging.Error("AppKiller crashed: %v", r)
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, false)
		logging.Error("Stack trace:\n%s", string(buf[:n]))
		dispose()
		os.Exit(1)
	}
}

func onReady() {
	systray.SetIcon(iconData)
	systray.SetTitle(applicationName)

	mLastAttempt := systray.AddMenuItem("Last attempt: initializing...", "Most recent attempt time")
	mQuit := systray.AddMenuItem("Exit", "Quit the AppKiller")
	mLastAttempt.Disable()

	// Function to update tooltip/menu with last attempt status
	updateAttempt := func(t time.Time, status string) {
		text := fmt.Sprintf("Last attempt: %s (%s)", t.Format("2006-01-02 15:04:05"), status)
		systray.SetTooltip(fmt.Sprintf("%s is running\n%s", applicationName, text))
		mLastAttempt.SetTitle(text)
	}

	go startAppKiller(updateAttempt)

	<-mQuit.ClickedCh
	systray.Quit()
}

func onExit() {
	logging.Info("Exiting AppKiller")
}

func startAppKiller(updateAttempt func(time.Time, string)) {
	defer handleCrash()

	c := cron.New()

	_, err := c.AddFunc(schedule, func() {
		status := killApp(appToKill)
		updateAttempt(time.Now(), status)
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to parse cron expression '%s': %v", schedule, err))
	}

	if atLaunch {
		logging.Info("Immediate execution requested")
		status := killApp(appToKill)
		updateAttempt(time.Now(), status)
	}

	c.Start()
	logging.Info("Cron schedule started with expression: %s", schedule)
	select {}
}

func killApp(name string) string {
	if name == "" {
		return "not required"
	}

	running, err := isProcessRunning(name)
	if err != nil {
		logging.Error("Failed to check if %s is running: %v", name, err)
		return "fail"
	}
	if !running {
		logging.Info("%s not running, no kill needed", name)
		return "not running"
	}

	cmd := exec.Command("taskkill", "/IM", name, "/F")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.SW_MINIMIZE, // Hide the console window
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		logging.Error("Failed to kill %s: %v\nOutput: %s", name, err, string(out))
		return "fail"
	}

	logging.Info("Killed process %s successfully", name)
	return "success"
}

func isProcessRunning(name string) (bool, error) {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq "+name)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.SW_MINIMIZE,
	}
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	// tasklist output header contains "Image Name", so check if process name appears beyond the header line
	return bytes.Contains(out, []byte(name)), nil
}
