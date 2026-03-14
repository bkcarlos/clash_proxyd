package assets

import (
	"fmt"
	"os"
	"path/filepath"
)

// Bundled reports whether any assets were compiled in.
func Bundled() bool {
	return len(MihomoBinary) > 0 || len(CountryMMDB) > 0
}

// Extract writes embedded assets to disk only when the target files are absent.
// binaryPath is the destination for the mihomo executable.
// mmdbPath is the destination for Country.mmdb.
// Does nothing and returns nil for stub (non-bundle) builds.
func Extract(binaryPath, mmdbPath string) error {
	if len(MihomoBinary) > 0 {
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(binaryPath), 0755); err != nil {
				return fmt.Errorf("assets: create binary dir: %w", err)
			}
			if err := os.WriteFile(binaryPath, MihomoBinary, 0755); err != nil {
				return fmt.Errorf("assets: write mihomo binary: %w", err)
			}
		}
	}

	if len(CountryMMDB) > 0 {
		if _, err := os.Stat(mmdbPath); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(mmdbPath), 0755); err != nil {
				return fmt.Errorf("assets: create mmdb dir: %w", err)
			}
			if err := os.WriteFile(mmdbPath, CountryMMDB, 0644); err != nil {
				return fmt.Errorf("assets: write Country.mmdb: %w", err)
			}
		}
	}

	return nil
}
