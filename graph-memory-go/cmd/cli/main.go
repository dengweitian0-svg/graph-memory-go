package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "graph-memory",
	Short: "Graph Memory - Knowledge Graph Context Engine",
	Long: `Graph Memory is a knowledge graph-based context engine that helps AI agents
overcome context explosion, cross-session forgetting, and skill isolation problems.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Graph Memory v1.0.0")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
