package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newResearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "research",
		Short: "Deep investigation with contrarian stress-testing",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Research mode coming soon.")
			return nil
		},
	}
}
