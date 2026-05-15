//go:build windows

package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

var (
	crypt32DLL           = windows.NewLazySystemDLL("Crypt32.dll")
	kernel32DLL          = windows.NewLazySystemDLL("Kernel32.dll")
	cryptProtectDataProc = crypt32DLL.NewProc("CryptProtectData")
	cryptUnprotectProc   = crypt32DLL.NewProc("CryptUnprotectData")
	localFreeProc        = kernel32DLL.NewProc("LocalFree")
)

const cryptProtectUIForbidden = 0x1

func (s *DesktopStore) protectSecret(value []byte) ([]byte, error) {
	return cryptProtect(value)
}

func (s *DesktopStore) unprotectSecret(value []byte) ([]byte, error) {
	return cryptUnprotect(value)
}

func cryptProtect(value []byte) ([]byte, error) {
	if len(value) == 0 {
		return []byte{}, nil
	}

	input := newDataBlob(value)
	var output dataBlob

	result, _, err := cryptProtectDataProc.Call(
		uintptr(unsafe.Pointer(&input)),
		0,
		0,
		0,
		0,
		uintptr(cryptProtectUIForbidden),
		uintptr(unsafe.Pointer(&output)),
	)
	if result == 0 {
		return nil, fmt.Errorf("CryptProtectData failed: %w", err)
	}

	return copyAndFreeDataBlob(output), nil
}

func cryptUnprotect(value []byte) ([]byte, error) {
	if len(value) == 0 {
		return []byte{}, nil
	}

	input := newDataBlob(value)
	var output dataBlob

	result, _, err := cryptUnprotectProc.Call(
		uintptr(unsafe.Pointer(&input)),
		0,
		0,
		0,
		0,
		uintptr(cryptProtectUIForbidden),
		uintptr(unsafe.Pointer(&output)),
	)
	if result == 0 {
		return nil, fmt.Errorf("CryptUnprotectData failed: %w", err)
	}

	return copyAndFreeDataBlob(output), nil
}

func newDataBlob(value []byte) dataBlob {
	if len(value) == 0 {
		return dataBlob{}
	}

	return dataBlob{
		cbData: uint32(len(value)),
		pbData: &value[0],
	}
}

func copyAndFreeDataBlob(blob dataBlob) []byte {
	if blob.cbData == 0 || blob.pbData == nil {
		return []byte{}
	}

	data := unsafe.Slice(blob.pbData, blob.cbData)
	copied := append([]byte(nil), data...)
	_, _, _ = localFreeProc.Call(uintptr(unsafe.Pointer(blob.pbData)))

	return copied
}
