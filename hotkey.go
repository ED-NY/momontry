//go:build windows

package main

import (
	"fmt"
	"strings"

	"github.com/lxn/win"
)

func formatHotkey(mods, key uint32) string {
	var parts []string
	if mods&modControl != 0 {
		parts = append(parts, "Ctrl")
	}
	if mods&modAlt != 0 {
		parts = append(parts, "Alt")
	}
	if mods&modShift != 0 {
		parts = append(parts, "Shift")
	}
	if mods&modWin != 0 {
		parts = append(parts, "Win")
	}
	if key != 0 {
		parts = append(parts, keyName(key))
	}
	if len(parts) == 0 {
		return T("unset")
	}
	return strings.Join(parts, "+")
}

func keyName(vk uint32) string {
	switch vk {
	case win.VK_SNAPSHOT:
		return "PrtSc"
	case win.VK_RETURN:
		return "Enter"
	case win.VK_SPACE:
		return "Space"
	case win.VK_DELETE:
		return "Delete"
	case win.VK_TAB:
		return "Tab"
	case win.VK_BACK:
		return "Backspace"
	case win.VK_LEFT:
		return "Left"
	case win.VK_RIGHT:
		return "Right"
	case win.VK_UP:
		return "Up"
	case win.VK_DOWN:
		return "Down"
	case vkPause:
		return "Pause"
	}
	if vk >= vkF1 && vk <= vkF24 {
		return fmt.Sprintf("F%d", vk-vkF1+1)
	}
	if vk >= '0' && vk <= '9' || vk >= 'A' && vk <= 'Z' {
		return string(rune(vk))
	}
	if vk >= win.VK_NUMPAD0 && vk <= win.VK_NUMPAD9 {
		return fmt.Sprintf("Num%d", vk-win.VK_NUMPAD0)
	}
	return fmt.Sprintf("VK%02X", vk)
}

func isModifierVK(vk uint32) bool {
	switch vk {
	case win.VK_SHIFT, win.VK_CONTROL, win.VK_MENU, win.VK_LWIN, vkRWin,
		vkLShift, vkRShift, vkLControl, vkRControl, vkLMenu, vkRMenu:
		return true
	default:
		return false
	}
}

func allowsBareKey(vk uint32) bool {
	if vk >= vkF1 && vk <= vkF24 {
		return true
	}
	return vk == uint32(win.VK_SNAPSHOT) || vk == vkPause
}

func currentModifiers() uint32 {
	var m uint32
	if win.GetKeyState(win.VK_CONTROL) < 0 || win.GetKeyState(vkLControl) < 0 || win.GetKeyState(vkRControl) < 0 {
		m |= modControl
	}
	if win.GetKeyState(win.VK_MENU) < 0 || win.GetKeyState(vkLMenu) < 0 || win.GetKeyState(vkRMenu) < 0 {
		m |= modAlt
	}
	if win.GetKeyState(win.VK_SHIFT) < 0 || win.GetKeyState(vkLShift) < 0 || win.GetKeyState(vkRShift) < 0 {
		m |= modShift
	}
	if win.GetKeyState(win.VK_LWIN) < 0 || win.GetKeyState(vkRWin) < 0 {
		m |= modWin
	}
	return m
}

func registerHotkey(mods, key uint32) error {
	procUnregisterHotKey.Call(uintptr(msgHwnd), hotkeyID)
	r, _, err := procRegisterHotKey.Call(uintptr(msgHwnd), hotkeyID, uintptr(mods|modNoRepeat), uintptr(key))
	if r == 0 {
		if err != nil && err.Error() != "The operation completed successfully." {
			return fmt.Errorf("%s", tf("hotkeyRegFail", err.Error()))
		}
		return fmt.Errorf("%s", T("hotkeyBusy"))
	}
	return nil
}
