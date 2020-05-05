package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xakep666/licensevalidator/cmd/licensevalidator/app"

	"github.com/urfave/cli/v2"
)

var (
	configFileFlag = cli.PathFlag{
		Name:    "config",
		Aliases: []string{"c"},
		Usage:   "Path to config file",
		Value:   "config.toml",
	}
)

// for mocking
var (
	args          = os.Args
	interruptChan = make(chan os.Signal, 1)
)

func main() {
	a := &cli.App{
		Name:   "License-Validator",
		Usage:  "Web service that validates go module license by query. Developed to be used as Athens admission web hook.",
		Action: action,
		Flags: []cli.Flag{
			&configFileFlag,
		},
		Commands: []*cli.Command{
			ConfigSampleCommand(),
		},
	}

	signal.Notify(interruptChan, syscall.SIGTERM, syscall.SIGINT)
	cli.HandleExitCoder(a.Run(args))
}

func action(ctx *cli.Context) error {
	fmt.Print(`
██╗     ██╗ ██████╗███████╗███╗   ██╗███████╗███████╗                 
██║     ██║██╔════╝██╔════╝████╗  ██║██╔════╝██╔════╝                 
██║     ██║██║     █████╗  ██╔██╗ ██║███████╗█████╗                   
██║     ██║██║     ██╔══╝  ██║╚██╗██║╚════██║██╔══╝                   
███████╗██║╚██████╗███████╗██║ ╚████║███████║███████╗                 
╚══════╝╚═╝ ╚═════╝╚══════╝╚═╝  ╚═══╝╚══════╝╚══════╝                 
                                                                      
██╗   ██╗ █████╗ ██╗     ██╗██████╗  █████╗ ████████╗ ██████╗ ██████╗ 
██║   ██║██╔══██╗██║     ██║██╔══██╗██╔══██╗╚══██╔══╝██╔═══██╗██╔══██╗
██║   ██║███████║██║     ██║██║  ██║███████║   ██║   ██║   ██║██████╔╝
╚██╗ ██╔╝██╔══██║██║     ██║██║  ██║██╔══██║   ██║   ██║   ██║██╔══██╗
 ╚████╔╝ ██║  ██║███████╗██║██████╔╝██║  ██║   ██║   ╚██████╔╝██║  ██║
  ╚═══╝  ╚═╝  ╚═╝╚══════╝╚═╝╚═════╝ ╚═╝  ╚═╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝
`)

	cfg, err := app.ConfigFromFile(ctx.Path(configFileFlag.Name))
	if err != nil {
		return cli.Exit(err, 1)
	}

	a, err := app.NewApp(cfg)
	if err != nil {
		return cli.Exit(fmt.Sprintf("Failed to init app: %s", err), 1)
	}

	errCh := make(chan error)

	go func() { errCh <- a.Run() }()

	select {
	case err := <-errCh:
		return cli.Exit(fmt.Sprintf("App run failed: %s", err), 2)
	case <-interruptChan:
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = a.Stop(stopCtx)
	if err != nil {
		return cli.Exit(fmt.Sprintf("Shutdown error: %s", err), 3)
	}

	return nil
}
