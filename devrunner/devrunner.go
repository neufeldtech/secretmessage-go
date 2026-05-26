package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func main() {
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		fmt.Println("Error: APP_URL environment variable is not set.")
		os.Exit(1)
	}

	shutdown := make(chan struct{})

	// Start ngrok
	ngrokCmd := exec.Command("ngrok", "http", "--url="+appURL, "8080")
	ngrokCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Start air
	airCmd := exec.Command("air")
	airCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Pipe output
	ngrokStdout, _ := ngrokCmd.StdoutPipe()
	ngrokStderr, _ := ngrokCmd.StderrPipe()
	airStdout, _ := airCmd.StdoutPipe()
	airStderr, _ := airCmd.StderrPipe()

	// Print helper
	printPrefixed := func(prefix string, color string, r io.Reader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			fmt.Printf("%s%s%s %s\n", color, prefix, "\033[0m", scanner.Text())
		}
	}

	// Start processes
	if err := ngrokCmd.Start(); err != nil {
		fmt.Printf("Failed to start ngrok: %v\n", err)
		os.Exit(1)
	}
	
	if err := airCmd.Start(); err != nil {
		fmt.Printf("Failed to start air: %v\n", err)
		// Kill ngrok
		syscall.Kill(-ngrokCmd.Process.Pid, syscall.SIGTERM)
		os.Exit(1)
	}

	// Goroutines to read logs concurrently
	go printPrefixed("[ngrok]", "\033[36m", ngrokStdout)
	go printPrefixed("[ngrok]", "\033[36m", ngrokStderr)
	go printPrefixed("[air]", "\033[32m", airStdout)
	go printPrefixed("[air]", "\033[32m", airStderr)

	// Listen for OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Goroutine to wait for either command to exit
	go func() {
		airCmd.Wait()
		fmt.Println("[devrunner] air process exited.")
		select {
		case <-shutdown:
		default:
			close(shutdown)
		}
	}()

	go func() {
		ngrokCmd.Wait()
		fmt.Println("[devrunner] ngrok process exited.")
		select {
		case <-shutdown:
		default:
			close(shutdown)
		}
	}()

	select {
	case sig := <-sigChan:
		fmt.Printf("[devrunner] Received signal: %v. Shutting down...\n", sig)
	case <-shutdown:
		fmt.Println("[devrunner] Subprocess exited. Shutting down...")
	}

	// Clean up process groups
	fmt.Println("[devrunner] Killing air process group...")
	syscall.Kill(-airCmd.Process.Pid, syscall.SIGINT)
	
	fmt.Println("[devrunner] Killing ngrok process group...")
	syscall.Kill(-ngrokCmd.Process.Pid, syscall.SIGTERM)
}
