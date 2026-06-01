package main

import (
	"os"
	"path/filepath"

	"github.com/urfave/cli"

	_ "github.com/DarthSim/godotenv/autoload"
)

// version is the build version. Overridden at release time via
// -ldflags "-X main.version=...". Defaults to the current pre-release.
var version = "1.2.0-rc.1"

func main() {
	var (
		conf logbeeConfig
		err  error
	)

	app := cli.NewApp()

	app.Name = "Logbee"
	app.HelpName = "logbee"
	app.Usage = "The mind to rule processes of your development environment"
	app.Description = "Logbee is a process manager for Procfile-based applications"
	app.Author = "neatnettech"
	app.Email = "pp@neatnet.tech"
	app.Version = version
	app.ArgsUsage = "[procfile] (Use '-' to read from stdin, Procfile path can be also set with $LOGBEE_PROCFILE)"
	app.HideHelp = true

	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "title, w", EnvVar: "LOGBEE_TITLE", Usage: "Specify a title of the application", Destination: &conf.Title},
		cli.StringFlag{Name: "processes, l", EnvVar: "LOGBEE_PROCESSES", Usage: "Specify process names to launch. Divide names with comma", Destination: &conf.ProcNames},
		cli.IntFlag{Name: "port, p", EnvVar: "LOGBEE_PORT,PORT", Usage: "specify a port to use as the base", Value: 5000, Destination: &conf.PortBase},
		cli.IntFlag{Name: "port-step, P", EnvVar: "LOGBEE_PORT_STEP", Usage: "specify a step to increase port number", Value: 100, Destination: &conf.PortStep},
		cli.StringFlag{Name: "root, d", EnvVar: "LOGBEE_ROOT", Usage: "specify a working directory of application. Default: directory containing the Procfile", Destination: &conf.Root},
		cli.IntFlag{Name: "timeout, t", EnvVar: "LOGBEE_TIMEOUT", Usage: "specify the amount of time (in seconds) processes have to shut down gracefully before being brutally killed", Value: 5, Destination: &conf.Timeout},
		cli.BoolFlag{Name: "no-prefix", EnvVar: "LOGBEE_NO_PREFIX", Usage: "process names will not be printed if the flag is specified", Destination: &conf.NoPrefix},
		cli.BoolFlag{Name: "print-timestamps, T", EnvVar: "LOGBEE_PRINT_TIMESTAMPS", Usage: "timestamps will be printed if the flag is specified", Destination: &conf.PrintTimestamps},
		cli.StringFlag{Name: "log-file, L", EnvVar: "LOGBEE_LOG_FILE", Usage: "write the aggregated, plain-text (no color) log stream to this file, live per line", Destination: &conf.LogFile},
		cli.BoolFlag{Name: "log-append", EnvVar: "LOGBEE_LOG_APPEND", Usage: "append to the log file instead of truncating it on start", Destination: &conf.LogAppend},
		cli.StringFlag{Name: "interactive, i", EnvVar: "LOGBEE_INTERACTIVE", Usage: "forward your terminal's stdin to this process so you can drive interactive dev servers (e.g. Expo/Metro: press r, i, a, j)", Destination: &conf.Interactive},
		cli.BoolFlag{Name: "tui, u", EnvVar: "LOGBEE_TUI", Usage: "interactive console: one tab per process plus an aggregate tab (requires a terminal)", Destination: &conf.TUI},
		cli.IntFlag{Name: "scrollback", EnvVar: "LOGBEE_SCROLLBACK", Usage: "number of log lines kept per tab in --tui mode", Value: 5000, Destination: &conf.Scrollback},
	}

	app.Action = func(c *cli.Context) error {
		switch c.NArg() {
		case 0:
			if path := os.Getenv("LOGBEE_PROCFILE"); len(path) > 0 {
				conf.Procfile = path
			} else {
				conf.Procfile = "./Procfile"
			}
		case 1:
			conf.Procfile = c.Args().First()
		default:
			fatal("Specify a single procfile")
		}

		if conf.Timeout < 1 {
			fatal("Timeout should be greater than 0")
		}

		if len(conf.Root) == 0 {
			conf.Root = filepath.Dir(conf.Procfile)
		}

		conf.Root, err = filepath.Abs(conf.Root)
		fatalOnErr(err)

		newLogbee(conf).Run()

		return nil
	}

	app.Run(os.Args)
}
