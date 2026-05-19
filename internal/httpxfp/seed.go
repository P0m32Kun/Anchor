package httpxfp

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/P0m32Kun/Anchor/internal/builtin"
	"github.com/P0m32Kun/Anchor/internal/models"
)

const builtinHttpxID = "builtin:rbkd-finger"

// SeedBuiltin upserts the RBKD-SEC/finger fingerprint row when finger.json exists
// under fingerRoot. Idempotent: refreshes description and file_path on each run;
// preserves Enabled if the user disabled the builtin row.
func (m *Manager) SeedBuiltin(fingerRoot string) error {
	fpPath := filepath.Join(fingerRoot, "finger.json")
	if _, err := os.Stat(fpPath); err != nil {
		if os.IsNotExist(err) {
			log.Printf("[httpxfp] seed: %s missing, skip", fpPath)
			return nil
		}
		return fmt.Errorf("stat %s: %w", fpPath, err)
	}

	absPath, err := filepath.Abs(fpPath)
	if err != nil {
		return fmt.Errorf("abs finger path: %w", err)
	}

	rev := builtin.HeadShort(fingerRoot)
	desc := "RBKD-SEC/finger"
	if rev != "" {
		desc = fmt.Sprintf("RBKD-SEC/finger @ %s", rev)
	}

	now := time.Now().UTC()
	f := &models.HttpxFingerprint{
		ID:          builtinHttpxID,
		Name:        "RBKD finger",
		Description: desc,
		Type:        models.HttpxFingerprintTypeTechDetect,
		FilePath:    absPath,
		Builtin:     true,
		Enabled:     true,
		UpdatedAt:   now,
	}

	existing, err := m.q.GetHttpxFingerprint(builtinHttpxID)
	if err != nil {
		return fmt.Errorf("get builtin fingerprint: %w", err)
	}
	if existing != nil {
		f.CreatedAt = existing.CreatedAt
		f.Enabled = existing.Enabled
		if err := m.q.UpdateHttpxFingerprint(f); err != nil {
			return fmt.Errorf("update builtin fingerprint: %w", err)
		}
		return nil
	}

	f.CreatedAt = now
	if err := m.q.CreateHttpxFingerprint(f); err != nil {
		return fmt.Errorf("create builtin fingerprint: %w", err)
	}
	return nil
}
