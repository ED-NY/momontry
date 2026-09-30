//go:build windows

package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/lxn/win"
)

var (
	hotDlg       win.HWND
	hotMods      uint32
	hotVK        uint32
	hotHint      string
	toastHwnd    win.HWND
	textHwnd     win.HWND
	textEdit     win.HWND
	textCopyBtn  win.HWND
	textCloseBtn win.HWND
	textBody     string
)

const (
	kindHotkey = 1
	kindToast  = 2
	kindText   = 3
)

func popupProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == win.WM_NCCREATE {
		setUserData(hwnd, nextPopupKind)
		return win.DefWindowProc(hwnd, msg, wParam, lParam)
	}
	switch getUserData(hwnd) {
	case kindHotkey:
		return hotkeyProc(hwnd, msg, wParam, lParam)
	case kindToast:
		return toastProc(hwnd, msg, wParam, lParam)
	case kindText:
		return textProc(hwnd, msg, wParam, lParam)
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func openHotkeyDialog() {
	if hotDlg != 0 {
		win.SetForegroundWindow(hotDlg)
		return
	}
	procUnregisterHotKey.Call(uintptr(msgHwnd), hotkeyID)
	c := getConfig()
	hotMods = c.Modifiers
	hotVK = c.Key
	hotHint = T("hotkeyPrompt")
	w, h := int32(460), int32(180)
	x, y := centerOnWork(w, h)
	nextPopupKind = kindHotkey
	hotDlg = win.CreateWindowEx(
		win.WS_EX_TOPMOST|win.WS_EX_TOOLWINDOW,
		classPopup.ptr,
		utf16p(T("hotkeyTitle")),
		win.WS_POPUP|win.WS_VISIBLE,
		x, y, w, h,
		0, 0, hinst, nil,
	)
	win.ShowWindow(hotDlg, win.SW_SHOW)
	win.SetForegroundWindow(hotDlg)
	win.SetFocus(hotDlg)
}

func closeHotkeyDialog(restore bool) {
	if hotDlg != 0 {
		h := hotDlg
		hotDlg = 0
		win.DestroyWindow(h)
	}
	if restore {
		c := getConfig()
		_ = registerHotkey(c.Modifiers, c.Key)
	}
}

func hotkeyProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_ERASEBKGND:
		return 1
	case win.WM_PAINT:
		paintHotkey(hwnd)
		return 0
	case win.WM_KEYDOWN, win.WM_SYSKEYDOWN:
		if lParam&(1<<30) != 0 {
			return 0
		}
		vk := uint32(wParam)
		if vk == uint32(win.VK_ESCAPE) {
			closeHotkeyDialog(true)
			return 0
		}
		if isModifierVK(vk) {
			hotMods = currentModifiers()
			hotVK = 0
			hotHint = formatHotkey(hotMods, 0)
			if hotVK == 0 {
				hotHint = stringsTrimMods(hotMods)
			}
			win.InvalidateRect(hwnd, nil, false)
			return 0
		}
		mods := currentModifiers()
		if mods == 0 && !allowsBareKey(vk) {
			hotHint = T("hotkeyNeedMod")
			hotMods = 0
			hotVK = 0
			win.InvalidateRect(hwnd, nil, false)
			return 0
		}
		if err := registerHotkey(mods, vk); err != nil {
			hotHint = err.Error()
			win.InvalidateRect(hwnd, nil, false)
			return 0
		}
		if err := updateConfig(func(c *Config) {
			c.Modifiers = mods
			c.Key = vk
		}); err != nil {
			logf("save hotkey: %v", err)
		}
		closeHotkeyDialog(false)
		refreshMenu()
		toast(tf("hotkeySet", formatHotkey(mods, vk)))
		return 0
	case win.WM_DESTROY:
		if hotDlg == hwnd {
			hotDlg = 0
		}
		return 0
	case win.WM_LBUTTONDOWN:
		win.SetFocus(hwnd)
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func stringsTrimMods(mods uint32) string {
	if mods == 0 {
		return T("hotkeyPrompt")
	}
	return formatHotkey(mods, 0)
}

func paintHotkey(hwnd win.HWND) {
	var ps win.PAINTSTRUCT
	hdc := win.BeginPaint(hwnd, &ps)
	var rc win.RECT
	win.GetClientRect(hwnd, &rc)
	fillRect(hdc, &rc, win.RGB(32, 32, 32))
	old := win.SelectObject(hdc, win.HGDIOBJ(fontUI))
	win.SetBkMode(hdc, win.TRANSPARENT)
	title := rc
	title.Top = 18
	title.Bottom = 48
	buf, _ := utf16Slice(T("hotkeyTitle"))
	win.SetTextColor(hdc, win.RGB(180, 180, 180))
	win.DrawTextEx(hdc, &buf[0], -1, &title, win.DT_CENTER|win.DT_SINGLELINE, nil)
	win.SelectObject(hdc, win.HGDIOBJ(fontBig))
	mid := rc
	mid.Top = 58
	mid.Bottom = 110
	label := hotHint
	if hotVK != 0 {
		label = formatHotkey(hotMods, hotVK)
	}
	lb, _ := utf16Slice(label)
	win.SetTextColor(hdc, win.RGB(255, 255, 255))
	win.DrawTextEx(hdc, &lb[0], -1, &mid, win.DT_CENTER|win.DT_SINGLELINE|win.DT_VCENTER, nil)
	win.SelectObject(hdc, win.HGDIOBJ(fontUI))
	foot := rc
	foot.Top = rc.Bottom - 36
	tip, _ := utf16Slice(T("escCancel"))
	win.SetTextColor(hdc, win.RGB(160, 160, 160))
	win.DrawTextEx(hdc, &tip[0], -1, &foot, win.DT_CENTER|win.DT_SINGLELINE, nil)
	win.SelectObject(hdc, old)
	win.EndPaint(hwnd, &ps)
}

func chooseSaveDir() {
	title := utf16p(T("chooseDir"))
	var name [260]uint16
	bi := win.BROWSEINFO{
		HwndOwner:      msgHwnd,
		PszDisplayName: &name[0],
		LpszTitle:      title,
		UlFlags:        0x0001 | 0x0040 | 0x0010,
	}
	pidl := win.SHBrowseForFolder(&bi)
	if pidl == 0 {
		return
	}
	defer win.CoTaskMemFree(pidl)
	pathBuf := make([]uint16, 32768)
	if !win.SHGetPathFromIDList(pidl, &pathBuf[0]) {
		alert(T("dirReadFail"))
		return
	}
	dir := utf16ToString(pathBuf)
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		alert(tf("dirUseFail", err.Error()))
		return
	}
	if err := updateConfig(func(c *Config) { c.SaveDir = dir }); err != nil {
		alert(tf("configSaveFail", err.Error()))
		return
	}
	refreshMenu()
	toast(T("dirUpdated"))
}

func writePNG(img *image.RGBA) (string, error) {
	c := getConfig()
	if c.SaveDir == "" {
		return "", fmt.Errorf("%s", T("noSaveDir"))
	}
	if err := os.MkdirAll(c.SaveDir, 0o755); err != nil {
		return "", err
	}
	name := time.Now().Format("momontry_20060102_150405.png")
	path := filepath.Join(c.SaveDir, name)
	if _, err := os.Stat(path); err == nil {
		name = time.Now().Format("momontry_20060102_150405.000.png")
		path = filepath.Join(c.SaveDir, name)
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return "", err
	}
	return path, nil
}

func toast(text string) {
	if toastHwnd != 0 {
		win.DestroyWindow(toastHwnd)
		toastHwnd = 0
	}
	toastText = text
	w, h := int32(420), int32(44)
	wa := workArea()
	x := wa.Left + (wa.Right-wa.Left-w)/2
	y := wa.Bottom - h - 48
	nextPopupKind = kindToast
	toastHwnd = win.CreateWindowEx(
		win.WS_EX_TOPMOST|win.WS_EX_TOOLWINDOW|wsExNoActivate,
		classPopup.ptr,
		titleApp.ptr,
		win.WS_POPUP|win.WS_VISIBLE,
		x, y, w, h,
		0, 0, hinst, nil,
	)
	win.SetTimer(toastHwnd, 1, 1600, 0)
	win.InvalidateRect(toastHwnd, nil, false)
}

var toastText string

func toastProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_ERASEBKGND:
		return 1
	case win.WM_PAINT:
		var ps win.PAINTSTRUCT
		hdc := win.BeginPaint(hwnd, &ps)
		var rc win.RECT
		win.GetClientRect(hwnd, &rc)
		fillRect(hdc, &rc, win.RGB(20, 20, 20))
		old := win.SelectObject(hdc, win.HGDIOBJ(fontUI))
		win.SetBkMode(hdc, win.TRANSPARENT)
		win.SetTextColor(hdc, win.RGB(255, 255, 255))
		buf, _ := utf16Slice(toastText)
		win.DrawTextEx(hdc, &buf[0], -1, &rc, win.DT_CENTER|win.DT_VCENTER|win.DT_SINGLELINE, nil)
		win.SelectObject(hdc, old)
		win.EndPaint(hwnd, &ps)
		return 0
	case win.WM_TIMER:
		win.KillTimer(hwnd, 1)
		if toastHwnd == hwnd {
			toastHwnd = 0
		}
		win.DestroyWindow(hwnd)
		return 0
	case win.WM_LBUTTONUP:
		win.KillTimer(hwnd, 1)
		if toastHwnd == hwnd {
			toastHwnd = 0
		}
		win.DestroyWindow(hwnd)
		return 0
	case win.WM_DESTROY:
		if toastHwnd == hwnd {
			toastHwnd = 0
		}
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func showTextWindow(title, body string) {
	if textHwnd != 0 {
		textBody = body
		setWindowText(textHwnd, title)
		setWindowText(textEdit, body)
		win.SetForegroundWindow(textHwnd)
		return
	}
	textBody = body
	w, h := int32(520), int32(360)
	x, y := centerOnWork(w, h)
	nextPopupKind = kindText
	textHwnd = win.CreateWindowEx(
		win.WS_EX_TOPMOST,
		classPopup.ptr,
		utf16p(title),
		wsOverlapped|win.WS_VISIBLE|wsClipChildren,
		x, y, w, h,
		0, 0, hinst, nil,
	)
	setWindowText(textHwnd, title)
	layoutTextWindow(textHwnd)
	win.ShowWindow(textHwnd, win.SW_SHOW)
	win.SetForegroundWindow(textHwnd)
}

func layoutTextWindow(hwnd win.HWND) {
	var rc win.RECT
	win.GetClientRect(hwnd, &rc)
	editW := rc.Right - 16
	editH := rc.Bottom - 64
	if editH < 40 {
		editH = 40
	}
	if textEdit == 0 {
		textEdit = win.CreateWindowEx(
			win.WS_EX_CLIENTEDGE,
			classEdit.ptr,
			nil,
			win.WS_CHILD|win.WS_VISIBLE|win.WS_VSCROLL|0x0004|0x0040|0x0800|0x1000,
			8, 8, editW, editH,
			hwnd, win.HMENU(1), hinst, nil,
		)
		win.SendMessage(textEdit, win.WM_SETFONT, uintptr(fontUI), 1)
		setWindowText(textEdit, textBody)
		mk := func(id uintptr, label string, x, w int32) win.HWND {
			b := win.CreateWindowEx(0, classButton.ptr, utf16p(label),
				win.WS_CHILD|win.WS_VISIBLE,
				x, rc.Bottom-44, w, 32,
				hwnd, win.HMENU(id), hinst, nil)
			win.SendMessage(b, win.WM_SETFONT, uintptr(fontUI), 1)
			return b
		}
		copyW, closeW := textButtonWidths()
		textCopyBtn = mk(idTextCopy, T("copyText"), 8, copyW)
		textCloseBtn = mk(idTextClose, T("close"), 16+copyW, closeW)
	}
	win.MoveWindow(textEdit, 8, 8, editW, editH, true)
	copyW, closeW := textButtonWidths()
	if textCopyBtn != 0 {
		win.MoveWindow(textCopyBtn, 8, rc.Bottom-44, copyW, 32, true)
	}
	if textCloseBtn != 0 {
		win.MoveWindow(textCloseBtn, 16+copyW, rc.Bottom-44, closeW, 32, true)
	}
}

func textButtonWidths() (copyW, closeW int32) {
	if currentLang() == "en" {
		return 140, 90
	}
	return 120, 90
}

func textProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_SIZE:
		layoutTextWindow(hwnd)
		return 0
	case win.WM_COMMAND:
		id := uint32(wParam) & 0xFFFF
		switch id {
		case idTextCopy:
			if err := copyText(textBody); err != nil {
				alert(err.Error())
			} else {
				toast(T("textCopied"))
			}
		case idTextClose:
			win.DestroyWindow(hwnd)
		}
		return 0
	case win.WM_CLOSE:
		win.DestroyWindow(hwnd)
		return 0
	case win.WM_DESTROY:
		textHwnd = 0
		textEdit = 0
		textCopyBtn = 0
		textCloseBtn = 0
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}
