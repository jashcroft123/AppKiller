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

const (
	applicationName  = "AppKiller"
	CREATE_NO_WINDOW = 0x08000000 // Hide cmd window when calling taskkill/tasklist
)

//go:embed icon.ico
var iconData []byte

var (
	appToKill string
	schedule  string
	atLaunch  bool
	silent    bool
	exCount   int
)

func hideConsole() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	freeConsole := kernel32.NewProc("FreeConsole")
	freeConsole.Call()
}

func init() {
	logging.Init()

	flag.StringVar(&appToKill, "app", "", "Name of the app/process to kill")
	flag.StringVar(&schedule, "cron", "*/10 * * * *", "Cron schedule (default: every 10 minutes)")
	flag.BoolVar(&atLaunch, "immediate", true, "Execute immediately on launch")
	flag.BoolVar(&silent, "silent", true, "Hide console on launch (silent mode)")
	flag.IntVar(&exCount, "count", -1, "Number of times to run the kill command (-1 for infinite)")
	flag.Parse()

	logging.Info("CLI args: %v", os.Args)

	if silent {
		hideConsole()
		logging.Info("Running in silent mode, console hidden.")
	}

	if appToKill == "" {
		// If no application is specified, we cannot proceed so log and exit
		logging.Error("No application specified to kill. Use -app <process_name> to specify the application.")
		os.Exit(1)
	}

	var err error
	mutexName := fmt.Sprintf("%sMutex_%s", applicationName, os.Getenv("USERNAME"))
	err = appMutex.CreateMutex(mutexName)
	if err != nil {
		logging.Error("Another instance is already running: %v", err)
		fmt.Println("AppKiller is already running.")
		os.Exit(1)
	} else {
		logging.Info("Mutex created successfully, proceeding with AppKiller initialization.")
	}

	logging.Info("AppKiller initialized with app='%s', cron='%s', atLaunch=%v, count=%d", appToKill, schedule, atLaunch, exCount)
}

func main() {
	defer handleCrash()
	defer cleanUp()
	systray.Run(onReady, onExit)
}

func cleanUp() {
	appMutex.ReleaseMutex()
	logging.Info("Mutex released, exiting AppKiller.")
	logging.Info("AppKiller exiting gracefully.")
	logging.Close()
}

func handleCrash() {
	if r := recover(); r != nil {
		logging.Error("AppKiller crashed: %v", r)
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, false)
		logging.Error("Stack trace:\n%s", string(buf[:n]))
		systray.Quit()
		os.Exit(1)
	}
}

func onReady() {
	systray.SetIcon(iconData)
	systray.SetTitle(applicationName)

	mLastAttempt := systray.AddMenuItem("Last attempt: initializing...", "Most recent attempt time")
	mShowLog := systray.AddMenuItem("Open Log", "Show the log file")
	mManual := systray.AddMenuItem("Trigger Now", "Manually trigger kill action")
	mQuit := systray.AddMenuItem("Exit", "Quit the AppKiller")
	mLastAttempt.Disable()

	updateAttempt := func(t time.Time, status string) {
		text := fmt.Sprintf("Last attempt: %s (%s)", t.Format("2006-01-02 15:04:05"), status)
		systray.SetTooltip(fmt.Sprintf("%s is running\n%s", applicationName, text))
		mLastAttempt.SetTitle(text)
		logging.Info("Last attempt at %s: %s", t.Format("2006-01-02 15:04:05"), status)
	}

	go startAppKiller(updateAttempt)

	// Manual trigger and quit listener
	go func() {
		for {
			select {
			case <-mManual.ClickedCh:
				logging.Info("Manual trigger clicked")
				status := killApp(appToKill)
				updateAttempt(time.Now(), status)
			case <-mShowLog.ClickedCh:
				logging.Info("Show log clicked")
				logging.Show()
				if err := logging.Show(); err != nil {
					logging.Error("Failed to open log file: %v", err)
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	logging.Info("Exiting AppKiller")
}

func startAppKiller(updateAttempt func(time.Time, string)) {
	defer handleCrash()

	c := cron.New()

	_, err := c.AddFunc(schedule, func() {
		logging.Info("Scheduled task triggered")
		status := killApp(appToKill)
		updateAttempt(time.Now(), status)
		if exCount > 0 {
			exCount--
			logging.Info("Remaining executions: %d", exCount)
			if exCount == 0 {
				logging.Info("Max count reached, stopping cron scheduler")
				c.Stop()
				systray.Quit() // just quit the whole app
			}
		}
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to parse cron expression '%s': %v", schedule, err))
	}

	if atLaunch {
		logging.Info("Immediate execution requested")
		killAppAction(appToKill, updateAttempt)
	}

	c.Start()
	logging.Info("Cron schedule started with expression: %s", schedule)
	select {}
}

func killAppAction(name string, updateAttempt func(time.Time, string)) {
	status := killApp(appToKill)
	updateAttempt(time.Now(), status)
}

func killApp(name string) string {
	if name == "" {
		return "App not specified"
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
		CreationFlags: CREATE_NO_WINDOW,
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
	cmd := exec.Command("tasklist")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: CREATE_NO_WINDOW,
	}
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	lines := bytes.Split(out, []byte{'\n'})
	nameLower := bytes.ToLower([]byte(name))
	for _, line := range lines {
		fields := bytes.Fields(line)
		if len(fields) > 0 && bytes.Equal(bytes.ToLower(fields[0]), nameLower) {
			return true, nil
		}
	}
	return false, nil
}
