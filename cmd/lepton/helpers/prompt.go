package helpers

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func Confirm(cmd *cobra.Command, message string) (bool, error) {
	message += "\nAre you sure you want to continue? [y/N] "
	_, err := fmt.Fprint(cmd.OutOrStdout(), message)
	if err != nil {
		return false, err
	}

	var confirm string
	_, err = fmt.Fscanf(cmd.InOrStdin(), "%s", &confirm)
	if err != nil {
		return false, err
	}
	return strings.ToLower(confirm) == "y", err
}
