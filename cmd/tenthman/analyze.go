package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAnalyzeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "analyze",
		Short: "Feed a document/decision for Tenth Man counter-analysis",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Analyze mode coming soon.")
			return nil
		},
	}
}
