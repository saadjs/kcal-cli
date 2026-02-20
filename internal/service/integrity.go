package service

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type BackupInfo struct {
	Path      string    `json:"path"`
	Checksum  string    `json:"checksum"`
	CreatedAt time.Time `json:"created_at"`
	SizeBytes int64     `json:"size_bytes"`
}

type DoctorReport struct {
	OrphanEntries      int `json:"orphan_entries"`
	InvalidMetadata    int `json:"invalid_metadata"`
	DuplicateEntryRows int `json:"duplicate_entry_rows"`
	FixedMetadataRows  int `json:"fixed_metadata_rows,omitempty"`
}

func CreateBackup(dbPath, outPath string) (BackupInfo, error) {
	if strings.TrimSpace(dbPath) == "" {
		return BackupInfo{}, fmt.Errorf("db path is required")
	}
	if strings.TrimSpace(outPath) == "" {
		return BackupInfo{}, fmt.Errorf("backup output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return BackupInfo{}, fmt.Errorf("create backup directory: %w", err)
	}
	if err := copyFile(dbPath, outPath); err != nil {
		return BackupInfo{}, err
	}
	checksum, err := fileSHA256(outPath)
	if err != nil {
		return BackupInfo{}, err
	}
	if err := os.WriteFile(outPath+".sha256", []byte(checksum+"\n"), 0o644); err != nil {
		return BackupInfo{}, fmt.Errorf("write checksum file: %w", err)
	}
	st, err := os.Stat(outPath)
	if err != nil {
		return BackupInfo{}, fmt.Errorf("stat backup: %w", err)
	}
	return BackupInfo{Path: outPath, Checksum: checksum, CreatedAt: st.ModTime(), SizeBytes: st.Size()}, nil
}

func RestoreBackup(backupPath, dbPath string, force bool) error {
	if strings.TrimSpace(backupPath) == "" || strings.TrimSpace(dbPath) == "" {
		return fmt.Errorf("backup path and db path are required")
	}
	if !force {
		if _, err := os.Stat(dbPath); err == nil {
			return fmt.Errorf("target db already exists; use --force to overwrite")
		}
	}
	checksumFile := backupPath + ".sha256"
	if expected, err := os.ReadFile(checksumFile); err == nil {
		actual, err := fileSHA256(backupPath)
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(expected)) != actual {
			return fmt.Errorf("backup checksum mismatch")
		}
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("create db directory: %w", err)
	}
	return copyFile(backupPath, dbPath)
}

func ListBackups(dir string) ([]BackupInfo, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read backup dir: %w", err)
	}
	out := make([]BackupInfo, 0)
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".db") {
			continue
		}
		full := filepath.Join(dir, f.Name())
		st, err := os.Stat(full)
		if err != nil {
			continue
		}
		checksum := ""
		if b, err := os.ReadFile(full + ".sha256"); err == nil {
			checksum = strings.TrimSpace(string(b))
		}
		out = append(out, BackupInfo{Path: full, Checksum: checksum, CreatedAt: st.ModTime(), SizeBytes: st.Size()})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func RunDoctor(db *sql.DB, fix bool) (DoctorReport, error) {
	report := DoctorReport{}
	if err := db.QueryRow(`SELECT COUNT(1) FROM entries e LEFT JOIN categories c ON c.id = e.category_id WHERE c.id IS NULL`).Scan(&report.OrphanEntries); err != nil {
		return report, fmt.Errorf("doctor orphan check: %w", err)
	}
	rows, err := db.Query(`SELECT id, IFNULL(metadata_json,'') FROM entries`)
	if err != nil {
		return report, fmt.Errorf("doctor metadata query: %w", err)
	}
	invalidIDs := make([]int64, 0)
	for rows.Next() {
		var id int64
		var meta string
		if err := rows.Scan(&id, &meta); err != nil {
			_ = rows.Close()
			return report, fmt.Errorf("doctor metadata scan: %w", err)
		}
		meta = strings.TrimSpace(meta)
		if meta == "" {
			continue
		}
		if !json.Valid([]byte(meta)) {
			report.InvalidMetadata++
			invalidIDs = append(invalidIDs, id)
		}
	}
	_ = rows.Close()

	if err := db.QueryRow(`
SELECT COALESCE(SUM(cnt-1),0) FROM (
  SELECT COUNT(*) AS cnt
  FROM entries e JOIN categories c ON c.id = e.category_id
  GROUP BY e.name, c.name, e.consumed_at, e.source_type
  HAVING cnt > 1
)
`).Scan(&report.DuplicateEntryRows); err != nil {
		return report, fmt.Errorf("doctor duplicate query: %w", err)
	}

	if fix && len(invalidIDs) > 0 {
		tx, err := db.Begin()
		if err != nil {
			return report, fmt.Errorf("doctor fix begin tx: %w", err)
		}
		for _, id := range invalidIDs {
			if _, err := tx.Exec(`UPDATE entries SET metadata_json = '', updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id); err != nil {
				_ = tx.Rollback()
				return report, fmt.Errorf("doctor fix metadata row %d: %w", id, err)
			}
			report.FixedMetadataRows++
		}
		if err := tx.Commit(); err != nil {
			return report, fmt.Errorf("doctor fix commit: %w", err)
		}
	}

	return report, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}
	if err := out.Sync(); err != nil {
		return fmt.Errorf("sync destination file: %w", err)
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file for checksum: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
