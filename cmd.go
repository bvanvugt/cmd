package main

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
	// External deps
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type CmdConfig struct {
	Env              map[string]string
	DevContainerName string
	Commands         map[string]*CmdCommand
}

type CmdCommand struct {
	Name  string
	Shell string
}

func main() {
	ctx := context.Background()

	config := initConfig()
	systemOut := color.New(color.FgBlack)

	rootCmd := &cobra.Command{
		Use: "cmd",
	}

	for n, c := range config.Commands {
		rootCmd.AddCommand(&cobra.Command{
			Use: n,
			Run: func(_ *cobra.Command, args []string) {
				startTime := time.Now()

				cmd := exec.CommandContext(ctx, "bash", "-c", c.Shell)

				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Stdin = os.Stdin

				go func() {
					c := make(chan os.Signal, 1)
					signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
					s := <-c

					systemOut.Printf("\nReceived signal: %s\n", s)
					err := cmd.Process.Signal(s)
					if err != nil {
						panic(err)
					}
				}()

				systemOut.Printf("Running [%s] at %s\n", c.Name, startTime.Format(time.Kitchen))
				cmd.Run()

				endTime := time.Now()
				systemOut.Printf("Completed [%s] at %s after %s\n", c.Name, endTime.Format(time.Kitchen), endTime.Sub(startTime))
			},
		})
	}

	_ = rootCmd.Execute()
}

func initConfig() *CmdConfig {
	viper.SetConfigName(".devcontainer/cmd")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	c := CmdConfig{
		DevContainerName: viper.GetString("devcontainer.name"),
		Env:              viper.GetStringMapString("env"),
		Commands:         make(map[string]*CmdCommand),
	}

	err = viper.UnmarshalKey("commands", &c.Commands)
	if err != nil {
		panic(err)
	}

	for name, command := range c.Commands {
		if command.Name == "" {
			command.Name = name
		}
	}

	return &c
}
