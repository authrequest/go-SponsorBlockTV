package main

import (
	"fmt"
	"os"

	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/config"
	"github.com/authrequest/go-SponsorBlockTV/internal/pkg/setup"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Initialize config
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	// Create and run the setup program
	p := tea.NewProgram(setup.InitialModel(cfg))
	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
