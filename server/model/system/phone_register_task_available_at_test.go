package system

import (
	"reflect"
	"testing"
)

func TestSysPhoneRegisterTaskHasAvailableAtField(t *testing.T) {
	field, ok := reflect.TypeOf(SysPhoneRegisterTask{}).FieldByName("AvailableAt")
	if !ok {
		t.Fatal("SysPhoneRegisterTask.AvailableAt field is missing")
	}
	if got, want := field.Tag.Get("json"), "availableAt"; got != want {
		t.Fatalf("AvailableAt json tag = %q, want %q", got, want)
	}
	if got := field.Tag.Get("gorm"); got == "" {
		t.Fatal("AvailableAt gorm tag is empty")
	}
}
