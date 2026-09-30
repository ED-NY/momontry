//go:build windows

package main

import "fmt"

// UI copy lives in zh and en. The tray item "语言 / Language" switches cfg.Lang.
// 界面文案放在 zh 和 en 里，托盘菜单「语言 / Language」会切换 cfg.Lang。

var zh = map[string]string{
	"shot":           "截屏",
	"shotTip":        "框选截屏",
	"hotkey":         "设置截屏快捷键",
	"hotkeyTip":      "修改全局快捷键",
	"hotkeyMenu":     "设置截屏快捷键（%s）",
	"saveDir":        "设置保存目录",
	"saveDirTip":     "选择截图保存位置",
	"autostart":      "开机自启",
	"autostartTip":   "登录 Windows 后自动运行",
	"language":       "语言：中文",
	"languageTip":    "切换为 English",
	"quit":           "退出",
	"quitTip":        "退出 Momontry",
	"tooltip":        "Momontry 截图  %s",
	"hotkeyFailHint": "%s\n请在托盘菜单里重新设置快捷键。",
	"autostartFail":  "设置开机自启失败: %s",
	"noDisplay":      "没有可用的显示器",
	"captureFail":    "截屏失败: %s",
	"captureErr":     "截屏失败",
	"bitmapFail":     "创建截图画面失败",
	"overlayFail":    "无法显示框选窗口",
	"copied":         "已复制到剪贴板",
	"saved":          "已保存 %s",
	"ocrWorking":     "正在识别文字…",
	"ocrTitle":       "文字识别",
	"ocrFail":        "识别失败：\r\n%s",
	"ocrEmpty":       "未识别到文字。\r\n请确认已安装 Windows 文字识别语言包（设置 → 时间和语言 → 语言）。",
	"ocrEmptyShort":  "未识别到文字。",
	"hint":           "拖拽框选区域    Enter 复制    Esc 取消    右键打开菜单",
	"btnCopy":        "复制",
	"btnPin":         "固定到桌面",
	"btnSave":        "保存",
	"btnOCR":         "提取文字",
	"btnCancel":      "取消",
	"menuCopy":       "复制到剪贴板",
	"menuPin":        "固定到桌面",
	"menuSave":       "保存到目录",
	"menuOCR":        "提取文字",
	"menuClose":      "关闭",
	"hotkeyPrompt":   "请按下新的截屏快捷键",
	"hotkeyNeedMod":  "字母和数字需要搭配 Ctrl、Alt、Shift 或 Win",
	"hotkeySet":      "快捷键已设置为 %s",
	"escCancel":      "Esc 取消",
	"hotkeyTitle":    "设置截屏快捷键",
	"chooseDir":      "选择截图保存目录",
	"dirReadFail":    "无法读取所选目录",
	"dirUseFail":     "无法使用该目录: %s",
	"configSaveFail": "保存配置失败: %s",
	"dirUpdated":     "保存目录已更新",
	"noSaveDir":      "未设置保存目录",
	"copyText":       "复制文字",
	"close":          "关闭",
	"textCopied":     "文字已复制",
	"pinFail":        "无法固定到桌面",
	"pinWindowFail":  "无法创建固定窗口",
	"pinnedHint":     "已固定：拖动移动，滚轮缩放，双击关闭，右键更多",
	"noImage":        "没有图像",
	"emptyImage":     "图像为空",
	"clipAlloc":      "分配剪贴板内存失败",
	"clipLock":       "锁定剪贴板内存失败",
	"clipOpen":       "无法打开剪贴板",
	"clipWrite":      "写入剪贴板失败",
	"hotkeyRegFail":  "注册快捷键失败: %s",
	"hotkeyBusy":     "注册快捷键失败，可能已被其他程序占用",
	"ocrTimeout":     "文字识别超时",
	"ocrPackMissing": "当前 Windows 没有可用的 OCR 语言包",
	"unset":          "未设置",
}

var en = map[string]string{
	"shot":           "Capture",
	"shotTip":        "Select a region",
	"hotkey":         "Set hotkey",
	"hotkeyTip":      "Change the global hotkey",
	"hotkeyMenu":     "Set hotkey (%s)",
	"saveDir":        "Save folder",
	"saveDirTip":     "Choose where screenshots are saved",
	"autostart":      "Start with Windows",
	"autostartTip":   "Run when you sign in",
	"language":       "Language: English",
	"languageTip":    "切换为中文",
	"quit":           "Quit",
	"quitTip":        "Quit Momontry",
	"tooltip":        "Momontry  %s",
	"hotkeyFailHint": "%s\nSet a new hotkey from the tray menu.",
	"autostartFail":  "Could not update startup setting: %s",
	"noDisplay":      "No display is available",
	"captureFail":    "Capture failed: %s",
	"captureErr":     "Capture failed",
	"bitmapFail":     "Could not prepare the screenshot",
	"overlayFail":    "Could not show the selection window",
	"copied":         "Copied to the clipboard",
	"saved":          "Saved %s",
	"ocrWorking":     "Reading text…",
	"ocrTitle":       "Text",
	"ocrFail":        "Could not read text:\r\n%s",
	"ocrEmpty":       "No text found.\r\nInstall a Windows OCR language pack under Settings → Time & language → Language & region.",
	"ocrEmptyShort":  "No text found.",
	"hint":           "Drag to select    Enter copies    Esc cancels    Right-click for menu",
	"btnCopy":        "Copy",
	"btnPin":         "Pin",
	"btnSave":        "Save",
	"btnOCR":         "Text",
	"btnCancel":      "Cancel",
	"menuCopy":       "Copy to clipboard",
	"menuPin":        "Pin to desktop",
	"menuSave":       "Save to folder",
	"menuOCR":        "Extract text",
	"menuClose":      "Close",
	"hotkeyPrompt":   "Press a new screenshot hotkey",
	"hotkeyNeedMod":  "Letters and digits need Ctrl, Alt, Shift, or Win",
	"hotkeySet":      "Hotkey set to %s",
	"escCancel":      "Esc cancels",
	"hotkeyTitle":    "Set hotkey",
	"chooseDir":      "Choose a folder for screenshots",
	"dirReadFail":    "Could not read the selected folder",
	"dirUseFail":     "Could not use that folder: %s",
	"configSaveFail": "Could not save settings: %s",
	"dirUpdated":     "Save folder updated",
	"noSaveDir":      "No save folder is set",
	"copyText":       "Copy text",
	"close":          "Close",
	"textCopied":     "Text copied",
	"pinFail":        "Could not pin to the desktop",
	"pinWindowFail":  "Could not create the pin window",
	"pinnedHint":     "Pinned: drag to move, scroll to resize, double-click to close, right-click for more",
	"noImage":        "No image",
	"emptyImage":     "Image is empty",
	"clipAlloc":      "Could not allocate clipboard memory",
	"clipLock":       "Could not lock clipboard memory",
	"clipOpen":       "Could not open the clipboard",
	"clipWrite":      "Could not write to the clipboard",
	"hotkeyRegFail":  "Could not register the hotkey: %s",
	"hotkeyBusy":     "Could not register the hotkey. It may already be in use.",
	"ocrTimeout":     "Text recognition timed out",
	"ocrPackMissing": "No Windows OCR language pack is available",
	"unset":          "Not set",
}

func currentLang() string {
	if getConfig().Lang == "en" {
		return "en"
	}
	return "zh"
}

func T(key string) string {
	table := zh
	if currentLang() == "en" {
		table = en
	}
	if s, ok := table[key]; ok {
		return s
	}
	if s, ok := zh[key]; ok {
		return s
	}
	return key
}

func tf(key string, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}
