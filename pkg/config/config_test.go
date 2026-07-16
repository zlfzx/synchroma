package config

import (
	"os"
	"path/filepath"
	"testing"

	"synchroma/pkg/models"
)

func TestLoadConfig(t *testing.T) {
	t.Run("load default profile", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		content := `{
			"profiles": {
				"default": {
					"source": {"Database":"mysql","Host":"localhost","Port":"3306","User":"root","Password":"pass","DBName":"src_db"},
					"target": {"Database":"mysql","Host":"remote","Port":"3306","User":"admin","Password":"secret","DBName":"tgt_db"}
				}
			}
		}`
		_ = os.WriteFile(configPath, []byte(content), 0600)

		src, tgt, err := LoadConfig(configPath, "default")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if src.Host != "localhost" {
			t.Errorf("expected source host 'localhost', got %q", src.Host)
		}
		if tgt.Host != "remote" {
			t.Errorf("expected target host 'remote', got %q", tgt.Host)
		}
		if src.DBName != "src_db" {
			t.Errorf("expected source DBName 'src_db', got %q", src.DBName)
		}
	})

	t.Run("load named profile", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		content := `{
			"profiles": {
				"default": {
					"source": {"Host":"localhost"},
					"target": {"Host":"localhost"}
				},
				"staging": {
					"source": {"Database":"postgres","Host":"staging-src","Port":"5432","User":"pg","Password":"pg","DBName":"staging_src"},
					"target": {"Database":"postgres","Host":"staging-tgt","Port":"5432","User":"pg","Password":"pg","DBName":"staging_tgt"}
				}
			}
		}`
		_ = os.WriteFile(configPath, []byte(content), 0600)

		src, tgt, err := LoadConfig(configPath, "staging")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if src.Database != "postgres" {
			t.Errorf("expected database 'postgres', got %q", src.Database)
		}
		if tgt.Host != "staging-tgt" {
			t.Errorf("expected target host 'staging-tgt', got %q", tgt.Host)
		}
	})

	t.Run("profile not found returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		content := `{"profiles": {"default": {"source": {}, "target": {}}}}`
		_ = os.WriteFile(configPath, []byte(content), 0600)

		_, _, err := LoadConfig(configPath, "production")
		if err == nil {
			t.Fatal("expected error for missing profile, got nil")
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		_, _, err := LoadConfig("/nonexistent/path.json", "default")
		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})

	t.Run("empty profile name defaults to 'default'", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		content := `{
			"profiles": {
				"default": {
					"source": {"Host":"default-host"},
					"target": {"Host":"default-host"}
				}
			}
		}`
		_ = os.WriteFile(configPath, []byte(content), 0600)

		src, _, err := LoadConfig(configPath, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if src.Host != "default-host" {
			t.Errorf("expected 'default-host', got %q", src.Host)
		}
	})
}

func TestLoadProfile(t *testing.T) {
	t.Run("loads filter fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		content := `{
			"profiles": {
				"default": {
					"source": {"Host":"localhost"},
					"target": {"Host":"localhost"},
					"exclude_tables": ["migrations", "sessions"],
					"include_tables": ["users"]
				}
			}
		}`
		_ = os.WriteFile(configPath, []byte(content), 0600)

		profile, err := LoadProfile(configPath, "default")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(profile.ExcludeTables) != 2 {
			t.Errorf("expected 2 exclude tables, got %d", len(profile.ExcludeTables))
		}
		if len(profile.IncludeTables) != 1 {
			t.Errorf("expected 1 include table, got %d", len(profile.IncludeTables))
		}
	})
}

func TestSaveConfig(t *testing.T) {
	t.Run("create new config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		src := models.DataSource{Database: "mysql", Host: "localhost", Port: "3306", User: "root", Password: "pass", DBName: "mydb"}
		tgt := models.DataSource{Database: "mysql", Host: "remote", Port: "3306", User: "admin", Password: "secret", DBName: "mydb_prod"}

		err := SaveConfig(configPath, "default", src, tgt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify by reading back
		loadedSrc, loadedTgt, err := LoadConfig(configPath, "default")
		if err != nil {
			t.Fatalf("failed to load saved config: %v", err)
		}
		if loadedSrc.Host != "localhost" {
			t.Errorf("expected source host 'localhost', got %q", loadedSrc.Host)
		}
		if loadedTgt.DBName != "mydb_prod" {
			t.Errorf("expected target DBName 'mydb_prod', got %q", loadedTgt.DBName)
		}
	})

	t.Run("append profile to existing config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		src1 := models.DataSource{Database: "mysql", Host: "localhost"}
		tgt1 := models.DataSource{Database: "mysql", Host: "remote1"}
		_ = SaveConfig(configPath, "default", src1, tgt1)

		src2 := models.DataSource{Database: "postgres", Host: "pg-host"}
		tgt2 := models.DataSource{Database: "postgres", Host: "pg-remote"}
		_ = SaveConfig(configPath, "staging", src2, tgt2)

		// Both profiles should exist
		_, _, err := LoadConfig(configPath, "default")
		if err != nil {
			t.Fatalf("default profile should exist: %v", err)
		}

		stagingSrc, _, err := LoadConfig(configPath, "staging")
		if err != nil {
			t.Fatalf("staging profile should exist: %v", err)
		}
		if stagingSrc.Database != "postgres" {
			t.Errorf("expected 'postgres', got %q", stagingSrc.Database)
		}
	})
}
