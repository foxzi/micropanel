package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "micropanel",
	Short: "MicroPanel - Static Hosting Control Panel",
	Long: `MicroPanel is a minimalist static hosting control panel.
Manages static websites with Nginx, SSL certificates via Let's Encrypt,
redirects, basic auth, and file editing.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
