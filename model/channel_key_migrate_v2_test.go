package model

import (
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestBuildLegacyChannelKeyMigrationQueryV2EscapesKeyColumnForMySQL(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "gorm:gorm@tcp(localhost:9910)/gorm?charset=utf8mb4&parseTime=True&loc=Local",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("failed to open dry-run mysql gorm db: %v", err)
	}

	stmt := buildLegacyChannelKeyMigrationQueryV2(db).Find(&[]Channel{}).Statement
	sql := stmt.SQL.String()

	if !strings.Contains(sql, "`key` <> ?") {
		t.Fatalf("expected escaped key column in SQL, got %q", sql)
	}
}
