package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const settingsURL = "https://hackatime.hackclub.com/my/wakatime_setup"

// opens a url in the default browser
// best effort so calling code can move on if the launch fails
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported os")
	}
	return cmd.Start()
}

// walks the user through pointing hackatime at their machine
// shows whats already installed opens the auth page waits for the user
// then re reads the config to confirm
func runSetup() (*config, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println()
	fmt.Println("  " + orange + "✦ looking for hackatime..." + reset)
	fmt.Println()

	cliPath := findWakatimeCLI()
	cfgFile, cfgExists := configFileFound()

	if cliPath != "" {
		fmt.Println("  " + green + "✓" + reset + " wakatime-cli: " + dim + cliPath + reset)
	} else {
		fmt.Println("  " + dim + "✗ wakatime-cli not found" + reset)
	}
	if cfgExists {
		fmt.Println("  " + green + "✓" + reset + " config file:  " + dim + cfgFile + reset)
		fmt.Println("  " + dim + "    (file exists but has no api_key. finish setup to add it.)" + reset)
	} else {
		fmt.Println("  " + dim + "✗ ~/.wakatime.cfg not found" + reset)
	}
	fmt.Println()
	fmt.Println("  " + txt + "finish your hackatime setup here:" + reset)
	fmt.Println("  " + bold + orange + settingsURL + reset)
	fmt.Println()
	fmt.Println("  " + dim + "-> the page walks you through installing wakatime-cli and writing ~/.wakatime.cfg." + reset)
	fmt.Println("  " + dim + "-> once done your editor plugins " + reset + bold + "and" + reset + dim + " hackfetch will both work." + reset)
	fmt.Println()
	fmt.Print("  " + orange + "open it in your browser? " + dim + "[Y/n] " + reset)

	resp, _ := reader.ReadString('\n')
	resp = strings.ToLower(strings.TrimSpace(resp))
	if resp == "" || resp == "y" || resp == "yes" {
		if err := openBrowser(settingsURL); err != nil {
			fmt.Println("  " + dim + "(browser open failed copy the url manually)" + reset)
		}
	}

	fmt.Println()
	fmt.Print("  " + orange + "press " + bold + "[enter]" + reset + orange + " when setup is done or " + bold + "q" + reset + orange + " to quit: " + reset)
	resp2, _ := reader.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(resp2)) == "q" {
		return nil, fmt.Errorf("canceled")
	}

	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("still no api_key in ~/.wakatime.cfg finish the setup page first then re run hackfetch")
	}
	fmt.Println()
	fmt.Println("  " + green + "✓ found your key in ~/.wakatime.cfg" + reset)
	fmt.Println()
	return cfg, nil
}
