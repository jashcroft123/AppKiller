# AppKiller

Go app to kill an app on a cron schedule

# Usage of .\appkiller.exe:

- `-app string`  
  Name of the app/process to kill

- `-count int`  
  Number of times to run the kill command (-1 for infinite) (default -1)

- `-cron string`  
  Cron schedule (default: every 10 minutes) (default "*/10 * * * *")

- `-immediate`  
  Execute immediately on launch (default true)

- `-silent`  
  Hide console on launch (silent mode) (default true)
