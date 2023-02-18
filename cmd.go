package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed files/*
var fs embed.FS

var devContainerCmd string

var systemOut, systemErr *color.Color

func init() {
	systemOut = color.New(color.FgHiBlack)
	systemErr = color.New(color.FgHiRed)
}

type CmdConfig struct {
	DevContainerName string
	DevContainerDir  string
	Env              map[string]string
	Commands         []*CmdCommand
}

type CmdCommand struct {
	Name  string
	Shell string
}

func (cc *CmdCommand) Run(_ *cobra.Command, args []string) {
	ctx := context.Background()

	startTime := time.Now()

	shellString := strings.Replace(cc.Shell, "$@", strings.Join(args, " "), 1)
	if len(devContainerCmd) > 0 {
		shellString = fmt.Sprintf("%s %s", devContainerCmd, shellString)
	}

	cmd := exec.CommandContext(ctx, "bash", "-c", shellString)

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

	err := cmd.Run()
	if err != nil {
		panic(err)
	}

	endTime := time.Now()
	systemOut.Printf("Completed [%s] at %s after %s\n", cc.Name, endTime.Format(time.Kitchen), endTime.Sub(startTime))
}

func writeFile(srcPath string, dstPath string) {
	if _, err := os.Stat(dstPath); err == nil {
		systemErr.Printf("File %s already exists\n", dstPath)
		return
	}

	srcData, err := fs.ReadFile(srcPath)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(dstPath, srcData, 0644)
	if err != nil {
		panic(err)
	}

	systemOut.Printf("Created file: %s\n", dstPath)
}

func main() {

	config := loadConfig()

	rootCmd := &cobra.Command{
		Use: "cmd",
	}

	initCmd := &cobra.Command{
		Use: "init",
		Run: func(cmd *cobra.Command, args []string) {
			_ = os.Mkdir(".devcontainer", os.ModePerm)
			writeFile("files/cmd.yaml", ".devcontainer/cmd.yaml")
		},
	}
	initCmd.AddCommand(&cobra.Command{
		Use: "go",
		Run: func(cmd *cobra.Command, args []string) {
			_ = os.Mkdir(".devcontainer", os.ModePerm)
			writeFile("files/go/cmd.yaml", ".devcontainer/cmd.yaml")
			writeFile("files/go/devcontainer.json", ".devcontainer/devcontainer.json")
			writeFile("files/go/devcontainer.env", ".devcontainer/devcontainer.env")
			writeFile("files/go/devcontainer.sh", ".devcontainer/devcontainer.sh")
		},
	})

	devCmd := &cobra.Command{
		Use: "dev",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			devContainerCmd = fmt.Sprintf("docker exec -it -w %s %s", config.DevContainerDir, config.DevContainerName)
		},
		Run: func(cmd *cobra.Command, args []string) {
			systemErr.Println("no command name provided")
		},
	}

	for _, c := range config.Commands {
		devCmd.AddCommand(&cobra.Command{
			Use: c.Name,
			Run: c.Run,
		})
		rootCmd.AddCommand(&cobra.Command{
			Use: c.Name,
			Run: c.Run,
		})
	}

	rootCmd.AddCommand(devCmd)
	rootCmd.AddCommand(initCmd)

	_ = rootCmd.Execute()
}

func loadConfig() *CmdConfig {
	c := CmdConfig{}

	viper.AddConfigPath(".devcontainer")
	viper.SetConfigName("cmd")
	viper.SetConfigType("yaml")

	err := viper.ReadInConfig()

	var x viper.ConfigFileNotFoundError
	if errors.As(err, &x) {
		// systemErr.Printf("Config file not found: .cmd.yaml\n")
		return &c
	}
	if err != nil {
		panic(err)
	}

	c.DevContainerName = viper.GetString("devcontainer.name")
	c.DevContainerDir = viper.GetString("devcontainer.dir")
	c.Env = viper.GetStringMapString("env")

	commands := viper.GetStringMapString("commands")
	for name, command := range commands {
		c.Commands = append(c.Commands, &CmdCommand{
			Name:  name,
			Shell: command,
		})
	}

	return &c
}
