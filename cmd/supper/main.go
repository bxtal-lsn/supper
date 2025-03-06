package main

import (
	"fmt"
	"os"

	"github.com/bxtal-lsn/supper/internal/ui/views"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Initialize our application
	p := tea.NewProgram(
		views.NewMainView(),
		tea.WithAltScreen(),       // Use the full terminal window
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Start the application
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running application: %v\n", err)
		os.Exit(1)
	}
}
