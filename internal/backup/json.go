package backup

import (
	"fmt"
	"os"
)

func ReadDB(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read db file: %w", err)
	}
	return data, nil
}

func WriteDB(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write db file: %w", err)
	}
	return nil
}
