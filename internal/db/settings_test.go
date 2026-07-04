package db

import (
	"testing"
)

func TestGetSystemSetting_Default(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	idIdx, err := db.GetSystemSetting(SettingIdentityKeyIndex)
	if err != nil {
		t.Fatalf("failed to get default identity key index: %v", err)
	}
	if idIdx != "0" {
		t.Errorf("expected default identity key index to be '0', got %q", idIdx)
	}
}

func TestSetSystemSetting_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	err := db.SetSystemSetting(SettingIdentityKeyIndex, "5")
	if err != nil {
		t.Fatalf("SetSystemSetting failed: %v", err)
	}
}

func TestGetSystemSetting_Updated(t *testing.T) {
	db := setupTestDB(t)
	defer db.conn.Close()

	err := db.SetSystemSetting(SettingIdentityKeyIndex, "5")
	if err != nil {
		t.Fatalf("SetSystemSetting failed: %v", err)
	}

	idIdx, err := db.GetSystemSetting(SettingIdentityKeyIndex)
	if err != nil {
		t.Fatalf("GetSystemSetting failed: %v", err)
	}
	if idIdx != "5" {
		t.Errorf("expected identity key index to be updated to '5', got %q", idIdx)
	}
}
