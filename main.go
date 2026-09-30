//go:build windows

package main

import (
	"errors"
	"runtime"
	"time"

	"github.com/getlantern/systray"
	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

var (
	instanceMutex windows.Handle
	miShot        *systray.MenuItem
	miHotkey      *systray.MenuItem
	miDir         *systray.MenuItem
	miAuto        *systray.MenuItem
	miLang        *systray.MenuItem
	miQuit        *systray.MenuItem
	hotkeyErr     error
)

func main() {
	enableDPI()
	loadConfig()
	if !acquireSingleton() {
		signalRunning()
		return
	}
	if getConfig().Autostart {
		if err := setAutostart(true); err != nil {
			logf("refresh autostart: %v", err)
		}
	}
	go uiMain()
	systray.Run(onReady, onExit)
}

func acquireSingleton() bool {
	name := utf16p(`Local\MomontrySingleton`)
	h, err := windows.CreateMutex(nil, false, name)
	runtime.KeepAlive(name)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return false
	}
	instanceMutex = h
	return true
}

func signalRunning() {
	for i := 0; i < 20; i++ {
		hwnd := win.FindWindow(classMsg.ptr, titleApp.ptr)
		if hwnd != 0 {
			win.PostMessage(hwnd, msgCapture, 0, 0)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func onReady() {
	<-uiReady
	systray.SetIcon(trayIcon())
	miShot = systray.AddMenuItem(T("shot"), T("shotTip"))
	miHotkey = systray.AddMenuItem(T("hotkey"), T("hotkeyTip"))
	miDir = systray.AddMenuItem(T("saveDir"), T("saveDirTip"))
	miAuto = systray.AddMenuItemCheckbox(T("autostart"), T("autostartTip"), getConfig().Autostart)
	miLang = systray.AddMenuItem(T("language"), T("languageTip"))
	systray.AddSeparator()
	miQuit = systray.AddMenuItem(T("quit"), T("quitTip"))
	refreshMenu()
	if hotkeyErr != nil {
		err := hotkeyErr
		invokeUI(func() {
			alert(tf("hotkeyFailHint", err.Error()))
		})
	}
	go menuLoop()
	c := getConfig()
	logf("ready hotkey=%s dir=%s", formatHotkey(c.Modifiers, c.Key), c.SaveDir)
}

func onExit() {
	if msgHwnd != 0 {
		win.PostMessage(msgHwnd, msgQuit, 0, 0)
	}
}

func menuLoop() {
	for {
		select {
		case <-miShot.ClickedCh:
			invokeUI(beginCapture)
		case <-miHotkey.ClickedCh:
			invokeUI(openHotkeyDialog)
		case <-miDir.ClickedCh:
			invokeUI(chooseSaveDir)
		case <-miAuto.ClickedCh:
			toggleAutostart()
		case <-miLang.ClickedCh:
			toggleLanguage()
		case <-miQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func toggleAutostart() {
	next := !getConfig().Autostart
	if err := setAutostart(next); err != nil {
		logf("autostart: %v", err)
		if getConfig().Autostart {
			miAuto.Check()
		} else {
			miAuto.Uncheck()
		}
		invokeUI(func() { alert(tf("autostartFail", err.Error())) })
		return
	}
	if err := updateConfig(func(c *Config) { c.Autostart = next }); err != nil {
		logf("save autostart: %v", err)
	}
	refreshMenu()
}

func toggleLanguage() {
	next := "en"
	if getConfig().Lang == "en" {
		next = "zh"
	}
	if err := updateConfig(func(c *Config) { c.Lang = next }); err != nil {
		logf("save language: %v", err)
	}
	refreshMenu()
	invokeUI(func() {
		if hotDlg != 0 {
			if hotVK == 0 {
				hotHint = T("hotkeyPrompt")
			}
			win.InvalidateRect(hotDlg, nil, false)
		}
		if textCopyBtn != 0 {
			setWindowText(textCopyBtn, T("copyText"))
		}
		if textCloseBtn != 0 {
			setWindowText(textCloseBtn, T("close"))
		}
		if textHwnd != 0 {
			setWindowText(textHwnd, T("ocrTitle"))
			layoutTextWindow(textHwnd)
		}
		if bar != nil && session != nil && session.selValid() {
			h := bar.hwnd
			bar = nil
			if h != 0 {
				win.DestroyWindow(h)
			}
			showToolbar()
		}
	})
}

func refreshMenu() {
	c := getConfig()
	label := formatHotkey(c.Modifiers, c.Key)
	if miShot != nil {
		miShot.SetTitle(T("shot"))
		miShot.SetTooltip(T("shotTip"))
	}
	if miHotkey != nil {
		miHotkey.SetTitle(tf("hotkeyMenu", label))
		miHotkey.SetTooltip(T("hotkeyTip"))
	}
	if miDir != nil {
		miDir.SetTitle(T("saveDir"))
		miDir.SetTooltip(c.SaveDir)
	}
	if miAuto != nil {
		miAuto.SetTitle(T("autostart"))
		miAuto.SetTooltip(T("autostartTip"))
		if c.Autostart {
			miAuto.Check()
		} else {
			miAuto.Uncheck()
		}
	}
	if miLang != nil {
		miLang.SetTitle(T("language"))
		miLang.SetTooltip(T("languageTip"))
	}
	if miQuit != nil {
		miQuit.SetTitle(T("quit"))
		miQuit.SetTooltip(T("quitTip"))
	}
	systray.SetTooltip(tf("tooltip", label))
}
