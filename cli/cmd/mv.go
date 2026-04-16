package cmd

import (
	"fmt"

	"github.com/SheltonZhu/115driver/cli/internal/output"
	"github.com/SheltonZhu/115driver/cli/internal/resolver"
	"github.com/spf13/cobra"
)

var mvCmd = &cobra.Command{
	Use:   "mv <source_path> <destination_dir>",
	Short: "Move files into a destination directory",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		srcPath := args[0]
		dstDir := args[1]

		fileID, _, err := resolver.ResolvePath(client, srcPath)
		if err != nil {
			return &exitError{code: output.ExitNotFound, msg: err.Error()}
		}

		dirID, err := resolver.ResolveDir(client, dstDir)
		if err != nil {
			return &exitError{code: output.ExitNotFound, msg: fmt.Sprintf("Destination directory not found: %s", dstDir)}
		}

		if err := client.Move(dirID, fileID); err != nil {
			return &exitError{code: output.ExitError, msg: err.Error()}
		}

		printer.PrintSuccess(map[string]interface{}{
			"source":          srcPath,
			"destination_dir": dstDir,
			"file_ids":        []string{fileID},
		})
		if !jsonOutput {
			fmt.Printf("Moved %s -> %s\n", srcPath, dstDir)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mvCmd)
}
