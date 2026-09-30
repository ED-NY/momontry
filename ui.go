//go:build windows

package main

import (
	"runtime"
	"runtime/debug"
	"sync"
	"unsafe"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

var nextPopupKind uintptr

var (
	hinst   win.HINSTANCE
	msgHwnd win.HWND
	fontUI  win.HFONT
	fontBig win.HFONT

	uiReady = make(chan struct{})

	invokeMu sync.Mutex
	invokeQ  []func()
)

func uiMain() {
	// Keep this goroutine on the OS thread that creates the windows and hotkey.
	// 窗口和热键必须始终留在创建它们的系统线程上，否则全屏窗口收不到绘制和鼠标消息，表现为黑屏卡死。
	runtime.LockOSThread()
	win.CoInitializeEx(nil, win.COINIT_APARTMENTTHREADED)
	hinst = win.GetModuleHandle(nil)
	fontUI = makeFont(-16, win.FW_NORMAL)
	fontBig = makeFont(-28, win.FW_SEMIBOLD)

	registerClass(classMsg.ptr, msgProc, 0, 0)
	registerClass(classOverlay.ptr, overlayProc, win.CS_HREDRAW|win.CS_VREDRAW, cursorCross())
	registerClass(classToolbar.ptr, toolbarProc, 0, cursorArrow())
	registerClass(classPin.ptr, pinProc, win.CS_DBLCLKS, cursorArrow())
	registerClass(classPopup.ptr, popupProc, 0, cursorArrow())

	msgHwnd = win.CreateWindowEx(
		win.WS_EX_TOOLWINDOW,
		classMsg.ptr,
		titleApp.ptr,
		0,
		0, 0, 0, 0,
		0, 0, hinst, nil,
	)
	if msgHwnd == 0 {
		logf("create message window failed")
	}

	c := getConfig()
	if err := registerHotkey(c.Modifiers, c.Key); err != nil {
		hotkeyErr = err
		logf("hotkey: %v", err)
	}
	close(uiReady)

	for {
		var msg win.MSG
		ret := win.GetMessage(&msg, 0, 0, 0)
		if ret == 0 || ret == -1 {
			return
		}
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}
}

func registerClass(name *uint16, proc interface{}, style uint32, cursor win.HCURSOR) {
	var wc win.WNDCLASSEX
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.Style = style
	wc.LpfnWndProc = windows.NewCallback(proc)
	wc.HInstance = hinst
	wc.HIcon = loadAppIcon(32)
	wc.HIconSm = loadAppIcon(16)
	wc.HCursor = cursor
	wc.HbrBackground = win.HBRUSH(win.GetStockObject(win.NULL_BRUSH))
	wc.LpszClassName = name
	win.RegisterClassEx(&wc)
}

func loadAppIcon(size int32) win.HICON {
	const lrShared = 0x00008000
	h := win.LoadImage(hinst, (*uint16)(unsafe.Pointer(uintptr(1))), win.IMAGE_ICON, size, size, lrShared)
	return win.HICON(h)
}

func makeFont(height int32, weight int32) win.HFONT {
	var lf win.LOGFONT
	lf.LfHeight = height
	lf.LfWeight = weight
	lf.LfCharSet = win.DEFAULT_CHARSET
	lf.LfQuality = win.CLEARTYPE_QUALITY
	face, _ := windows.UTF16FromString("Microsoft YaHei UI")
	copy(lf.LfFaceName[:], face)
	return win.CreateFontIndirect(&lf)
}

func cursorCross() win.HCURSOR {
	return win.LoadCursor(0, (*uint16)(unsafe.Pointer(uintptr(win.IDC_CROSS))))
}

func cursorArrow() win.HCURSOR {
	return win.LoadCursor(0, (*uint16)(unsafe.Pointer(uintptr(win.IDC_ARROW))))
}

func invokeUI(fn func()) {
	invokeMu.Lock()
	invokeQ = append(invokeQ, fn)
	invokeMu.Unlock()
	if msgHwnd != 0 {
		win.PostMessage(msgHwnd, msgInvoke, 0, 0)
	}
}

func drainInvoke() {
	for {
		invokeMu.Lock()
		if len(invokeQ) == 0 {
			invokeMu.Unlock()
			return
		}
		fn := invokeQ[0]
		invokeQ = invokeQ[1:]
		invokeMu.Unlock()
		func() {
			defer func() {
				if r := recover(); r != nil {
					logf("ui panic: %v\n%s", r, debug.Stack())
				}
			}()
			fn()
		}()
	}
}

func msgProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_HOTKEY:
		if wParam == hotkeyID {
			beginCapture()
		}
		return 0
	case msgCapture:
		beginCapture()
		return 0
	case msgInvoke:
		drainInvoke()
		return 0
	case msgQuit:
		procUnregisterHotKey.Call(uintptr(msgHwnd), hotkeyID)
		destroyCapture()
		win.PostQuitMessage(0)
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func alert(text string) {
	body := utf16p(text)
	title := utf16p("Momontry")
	owner := msgHwnd
	win.MessageBox(owner, body, title, win.MB_OK|win.MB_ICONINFORMATION)
}
