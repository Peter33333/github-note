package open

import (
	"fmt"
	"os/exec"
)

func URL(target string) error {
	if target == "" {
		return fmt.Errorf("empty url")
	}
	cmd := exec.Command("xdg-open", target)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open url: %w", err)
	}
	return nil
}
