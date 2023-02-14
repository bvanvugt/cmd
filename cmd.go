package main

import (
	"context"
	"errors"
	"fmt"
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

var devContainerCmd string

var systemOut, systemErr *color.Color

type CmdConfig struct {
	DevContainerName string
	DevContainerDir  string
	Env              map[string]string
	Commands         map[string]*CmdCommand
}

type CmdCommand struct {
	Name  string
	Shell string
}

func (cc *CmdCommand) Run(_ *cobra.Command, args []string) {
	ctx := context.Background()

	startTime := time.Now()

	cmd := exec.CommandContext(ctx, "bash", "-c", fmt.Sprintf("%s %s", devContainerCmd, cc.Shell))

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

	systemOut.Printf("Running [%s] at %s\n", cc.Name, startTime.Format(time.Kitchen))
	cmd.Run()

	endTime := time.Now()
	systemOut.Printf("Completed [%s] at %s after %s\n", cc.Name, endTime.Format(time.Kitchen), endTime.Sub(startTime))
}

func init() {
	systemOut = color.New(color.FgHiBlack)
	systemErr = color.New(color.FgHiRed)
}

func main() {

	config := loadConfig()

	rootCmd := &cobra.Command{
		Use: "cmd",
	}

	devCmd := &cobra.Command{
		Use: "dev",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			devContainerCmd = fmt.Sprintf("docker exec -it -w %s %s", config.DevContainerDir, config.DevContainerName)
		},
		Run: func(cmd *cobra.Command, args []string) {
			systemErr.Println("no command name provided")
		},
	}

	for n, c := range config.Commands {
		devCmd.AddCommand(&cobra.Command{
			Use: n,
			Run: c.Run,
		})
		rootCmd.AddCommand(&cobra.Command{
			Use: n,
			Run: c.Run,
		})
	}

	rootCmd.AddCommand(devCmd)

	_ = rootCmd.Execute()
}

func loadConfig() *CmdConfig {
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	viper.SetConfigName(".cmd")

	err := viper.ReadInConfig()

	var x viper.ConfigFileNotFoundError
	if errors.As(err, &x) {
		systemErr.Printf("Config file not found: .cmd.yaml")
		os.Exit(1)
	}
	if err != nil {
		panic(err)
	}

	c := CmdConfig{
		DevContainerName: viper.GetString("devcontainer.name"),
		DevContainerDir:  viper.GetString("devcontainer.dir"),
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
