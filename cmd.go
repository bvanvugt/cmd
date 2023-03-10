package main

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

const DOCKER_PATH = "/usr/local/bin/docker"

type Config struct {
	DevContainerName string
	DevContainerDir  string
	Env              map[string]string
	Commands         map[string]string
}

func loadConfig() *Config {
	c := Config{}

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
	c.Commands = viper.GetStringMapString("commands")

	return &c
}

func main() {
	runInDevContainer := false

	systemOut := color.New(color.FgHiBlack)
	systemErr := color.New(color.FgHiRed)

	config := loadConfig()

	// Does command exist?
	cmdArg := os.Args[1]
	cmdArgs := os.Args[2:]
	if cmdArg == "dev" {
		runInDevContainer = true
		cmdArg = os.Args[2]
		cmdArgs = os.Args[3:]
	}

	cmdStr, found := config.Commands[cmdArg]
	if !found {
		systemErr.Println("command not found")
		return
	}

	// Create temp .sh file
	f, err := ioutil.TempFile(".devcontainer", "cmd.*.sh")
	if err != nil {
		panic(err)
	}
	defer os.Remove(f.Name())

	// Make it executable
	err = os.Chmod(f.Name(), 0755)
	if err != nil {
		panic(err)
	}

	// Write command to file
	_, err = f.WriteString("#!/bin/bash\n\n" + cmdStr + "\n")
	if err != nil {
		panic(err)
	}

	// Build command to run
	var cmd exec.Cmd
	if runInDevContainer {

		// Build docker exec command
		if len(config.DevContainerName) <= 0 {
			systemErr.Println("cmd.yaml: devcontainer.name not specified")
		}

		args := []string{"exec", "-it"}
		if len(config.DevContainerDir) > 0 {
			args = append(args, "-w", config.DevContainerDir)
		}
		args = append(
			args,
			config.DevContainerName,
			".devcontainer/"+path.Base(f.Name()))
		args = append(args, cmdArgs...)

		cmd = exec.Cmd{
			Path: DOCKER_PATH,
			Args: append([]string{DOCKER_PATH}, args...),
		}
	} else {
		// Build local bash command
		cmd = exec.Cmd{
			Path: f.Name(),
			Args: append([]string{f.Name()}, cmdArgs...),
		}

	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start listening for signals
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		s := <-c

		err := cmd.Process.Signal(s)
		if err != nil {
			panic(err)
		}
	}()

	// Run it
	startTime := time.Now()
	systemOut.Printf("Running [%s] at %s\n", cmdArg, startTime.Format(time.Kitchen))
	// systemOut.Println(cmd.String())
	err = cmd.Run()
	if err != nil {
		// Interrupted -- do we care?
		systemOut.Println(err)
		// log.Fatal(err)
	}
	endTime := time.Now()
	systemOut.Printf("Completed [%s] at %s after %s\n", cmdArg, endTime.Format(time.Kitchen), endTime.Sub(startTime))
}
