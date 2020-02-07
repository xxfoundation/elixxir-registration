////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

//go:generate go run gen.go
// The above generates: GITVERSION, DEPENDENCIES, and SEMVER

func main() {
	GenerateVersionFile()
}

func printVersion() {
	fmt.Printf("Elixxir Registration Server v%s -- %s\n\n", SEMVER, GITVERSION)
	fmt.Printf("Dependencies:\n\n%s\n", DEPENDENCIES)
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Elixxir binary",
	Long:  `Print the version number of Elixxir binary. This also prints the go mod dependencies file.`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}
