package x10

import (
    "testing"
)

func TestStringToAddress(t *testing.T) {
	a, err := StringToAddress("A1")
	if err != nil || a != 204 {
		t.Error("Invalid convert A1")
	}

	a, err = StringToAddress("a1")
	if err != nil || a != 204 {
		t.Error("Invalid convert a1")
	}
	
	_, err = StringToAddress("")
	if err == nil {
		t.Error("Invalid convert empty string")
	}
	
	_, err = StringToAddress("Z1")
	if err == nil {
		t.Error("Invalid convert Z1")
	}

	_, err = StringToAddress("A125")
	if err == nil {
		t.Error("Invalid convert A125")
	}
	
}
