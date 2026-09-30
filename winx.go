//go:build windows

package main

import (
	"unsafe"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

const (
	modAlt      = 0x0001
	modControl  = 0x0002
	modShift    = 0x0004
	modWin      = 0x0008
	modNoRepeat = 0x4000

	wsExNoActivate = 0x08000000
	wsClipChildren = 0x02000000
	wsOverlapped   = 0x00CF0000 // WS_OVERLAPPEDWINDOW

	swpNoActivate = 0x0010

	gwlpUserdata = ^uintptr(20) // -21

	spiGetWorkArea = 0x0030

	cfUnicodeText = 13

	wmNull = 0x0000
	wmApp  = 0x8000

	msgInvoke  = wmApp + 1
	msgCapture = wmApp + 2
	msgQuit    = wmApp + 3

	hotkeyID = 1

	vkRWin     = 0x5C
	vkLShift   = 0xA0
	vkRShift   = 0xA1
	vkLControl = 0xA2
	vkRControl = 0xA3
	vkLMenu    = 0xA4
	vkRMenu    = 0xA5
	vkF1       = 0x70
	vkF24      = 0x87
	vkPause    = 0x13

	actCopy   = 1
	actPin    = 2
	actSave   = 3
	actOCR    = 4
	actCancel = 5

	idTextCopy  = 2001
	idTextClose = 2002
)

var (
	user32 = windows.NewLazySystemDLL("user32.dll")
	gdi32  = windows.NewLazySystemDLL("gdi32.dll")

	procRegisterHotKey                = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey              = user32.NewProc("UnregisterHotKey")
	procAppendMenu                    = user32.NewProc("AppendMenuW")
	procFillRect                      = user32.NewProc("FillRect")
	procSetWindowText                 = user32.NewProc("SetWindowTextW")
	procSetWindowLongPtr              = user32.NewProc("SetWindowLongPtrW")
	procGetWindowLongPtr              = user32.NewProc("GetWindowLongPtrW")
	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	procCreatePen                     = gdi32.NewProc("CreatePen")
	procCreateSolidBrush              = gdi32.NewProc("CreateSolidBrush")
	procRectangle                     = gdi32.NewProc("Rectangle")
)

type wide struct {
	buf []uint16
	ptr *uint16
}

func wstr(s string) wide {
	buf, err := windows.UTF16FromString(s)
	if err != nil {
		buf = []uint16{0}
	}
	return wide{buf: buf, ptr: &buf[0]}
}

func utf16p(s string) *uint16 {
	p, err := windows.UTF16PtrFromString(s)
	if err != nil {
		return nil
	}
	return p
}

var (
	classMsg     = wstr("MomontryMsg")
	classOverlay = wstr("MomontryOverlay")
	classToolbar = wstr("MomontryToolbar")
	classPin     = wstr("MomontryPin")
	classPopup   = wstr("MomontryPopup")
	titleApp     = wstr("Momontry")
	classEdit    = wstr("EDIT")
	classButton  = wstr("BUTTON")
)

func enableDPI() {
	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 = (HANDLE)-4
	r, _, _ := procSetProcessDpiAwarenessContext.Call(^uintptr(3))
	if r == 0 {
		shcore := windows.NewLazySystemDLL("shcore.dll")
		set := shcore.NewProc("SetProcessDpiAwareness")
		if err := set.Find(); err == nil {
			_, _, _ = set.Call(2)
		}
	}
}

func setUserData(hwnd win.HWND, v uintptr) {
	procSetWindowLongPtr.Call(uintptr(hwnd), gwlpUserdata, v)
}

func getUserData(hwnd win.HWND) uintptr {
	r, _, _ := procGetWindowLongPtr.Call(uintptr(hwnd), gwlpUserdata)
	return r
}

func setWindowText(hwnd win.HWND, text string) {
	p := utf16p(text)
	if p == nil {
		return
	}
	procSetWindowText.Call(uintptr(hwnd), uintptr(unsafe.Pointer(p)))
}

func fillRect(hdc win.HDC, r *win.RECT, color win.COLORREF) {
	brush, _, _ := procCreateSolidBrush.Call(uintptr(color))
	if brush == 0 {
		return
	}
	procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(r)), brush)
	win.DeleteObject(win.HGDIOBJ(brush))
}

func appendMenuItem(menu win.HMENU, flags uintptr, id uintptr, text string) {
	if text == "" {
		procAppendMenu.Call(uintptr(menu), flags, id, 0)
		return
	}
	p := utf16p(text)
	procAppendMenu.Call(uintptr(menu), flags, id, uintptr(unsafe.Pointer(p)))
}

func workArea() win.RECT {
	var rc win.RECT
	win.SystemParametersInfo(spiGetWorkArea, 0, unsafe.Pointer(&rc), 0)
	if rc.Right <= rc.Left || rc.Bottom <= rc.Top {
		rc.Right = win.GetSystemMetrics(win.SM_CXSCREEN)
		rc.Bottom = win.GetSystemMetrics(win.SM_CYSCREEN)
	}
	return rc
}

func centerOnWork(w, h int32) (int32, int32) {
	wa := workArea()
	x := wa.Left + (wa.Right-wa.Left-w)/2
	y := wa.Top + (wa.Bottom-wa.Top-h)/2
	return x, y
}

func utf16Slice(s string) ([]uint16, error) {
	if s == "" {
		return []uint16{0}, nil
	}
	return windows.UTF16FromString(s)
}

func utf16ToString(buf []uint16) string {
	return windows.UTF16ToString(buf)
}
