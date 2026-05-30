package response

import (
	"reflect"
	"testing"
)

func TestPhoneRegisterTaskActiveInfoHasAvailableAtField(t *testing.T) {
	field, ok := reflect.TypeOf(PhoneRegisterTaskActiveInfo{}).FieldByName("AvailableAt")
	if !ok {
		t.Fatal("PhoneRegisterTaskActiveInfo.AvailableAt field is missing")
	}
	if got, want := field.Tag.Get("json"), "availableAt,omitempty"; got != want {
		t.Fatalf("AvailableAt json tag = %q, want %q", got, want)
	}
}
