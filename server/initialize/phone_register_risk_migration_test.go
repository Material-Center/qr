package initialize

import (
	"os"
	"strings"
	"testing"
)

func TestPhoneRegisterRiskDailyStatIsAutoMigrated(t *testing.T) {
	gormSource, err := os.ReadFile("gorm.go")
	if err != nil {
		t.Fatalf("read gorm.go: %v", err)
	}
	if !strings.Contains(string(gormSource), "system.SysPhoneRegisterRiskDailyStat{}") {
		t.Fatal("RegisterTables must auto-migrate system.SysPhoneRegisterRiskDailyStat")
	}

	ensureSource, err := os.ReadFile("ensure_tables.go")
	if err != nil {
		t.Fatalf("read ensure_tables.go: %v", err)
	}
	if got := strings.Count(string(ensureSource), "sysModel.SysPhoneRegisterRiskDailyStat{}"); got < 2 {
		t.Fatalf("ensureTables must migrate and check sysModel.SysPhoneRegisterRiskDailyStat, got %d references", got)
	}
}
