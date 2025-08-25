package main

import (
	"beszel"
	"beszel/internal/agent"
	"beszel/internal/agent/health"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

// cli options
type cmdOptions struct {
	key string // key is the base64 authentication key for hub verification.
}

// parse parses the command line flags and populates the config struct.
// It returns true if a subcommand was handled and the program should exit.
func (opts *cmdOptions) parse() bool {
	flag.StringVar(&opts.key, "key", "", "Base64 authentication key for hub verification")

	flag.Usage = func() {
		builder := strings.Builder{}
		builder.WriteString("Usage: ")
		builder.WriteString(os.Args[0])
		builder.WriteString(" [command] [flags]\n")
		builder.WriteString("\nCommands:\n")
		builder.WriteString("  health    Check if the agent is running\n")
		builder.WriteString("  help      Display this help message\n")
		builder.WriteString("  update    Update to the latest version\n")
		builder.WriteString("\nFlags:\n")
		fmt.Print(builder.String())
		flag.PrintDefaults()
	}

	subcommand := ""
	if len(os.Args) > 1 {
		subcommand = os.Args[1]
	}

	switch subcommand {
	case "-v", "version":
		fmt.Println(beszel.AppName+"-agent", beszel.Version)
		return true
	case "help":
		flag.Usage()
		return true
	case "update":
		agent.Update()
		return true
	case "health":
		err := health.Check()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print("ok")
		return true
	}

	flag.Parse()
	return false
}

// loadAuthKey loads the base64 authentication key from the command line flag, environment variable, or key file.
func (opts *cmdOptions) loadAuthKey() (string, error) {
	var keyData string

	// Try command line flag first
	if opts.key != "" {
		keyData = opts.key
	} else if key, ok := agent.GetEnv("KEY"); ok && key != "" {
		// Try environment variable
		keyData = key
	} else if keyFile, ok := agent.GetEnv("KEY_FILE"); ok {
		// Try key file
		pubKey, err := os.ReadFile(keyFile)
		if err != nil {
			return "", fmt.Errorf("failed to read key file: %w", err)
		}
		keyData = string(pubKey)
	} else {
		return "", fmt.Errorf("no authentication key provided: must set -key flag, KEY env var, or KEY_FILE env var. Use 'beszel-agent help' for usage")
	}

	// Ensure the key has the base64: prefix
	if !strings.HasPrefix(keyData, "base64:") {
		keyData = "base64:" + keyData
	}

	return keyData, nil
}

func main() {
	var opts cmdOptions
	subcommandHandled := opts.parse()

	if subcommandHandled {
		return
	}

	var serverConfig agent.ServerOptions
	var err error
	serverConfig.AuthKey, err = opts.loadAuthKey()
	if err != nil {
		log.Fatal("Failed to load authentication key:", err)
	}

	a, err := agent.NewAgent()
	if err != nil {
		log.Fatal("Failed to create agent: ", err)
	}

	if err := a.Start(serverConfig); err != nil {
		log.Fatal("Failed to start agent: ", err)
	}
}
