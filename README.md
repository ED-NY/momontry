# Momontry

Windows tray screenshot tool. It stays in the notification area and does not take a taskbar button.

Windows 托盘截图工具。启动后只留在通知区域，不占用任务栏。

The default hotkey is **Ctrl+Alt+A**. Starting the program again while it is already running begins a capture.

默认快捷键是 **Ctrl+Alt+A**。再次启动已在运行的程序会直接开始截屏。

## Use / 使用

- Right-click or left-click the tray icon: Capture, set hotkey, save folder, start with Windows, language, quit.
- 托盘图标右键（或左键）打开菜单：截屏、设置快捷键、设置保存目录、开机自启、语言、退出。
- **语言：中文** switches the interface to English. **Language: English** switches it back to Chinese. The choice is saved in the config file.
- 菜单里的「语言：中文」会把界面切到英文，「Language: English」会切回中文，并写入配置。
- Drag to select a region. `Enter` copies it, `Esc` cancels.
- 按下快捷键后拖拽框选。`Enter` 复制到剪贴板，`Esc` 取消。
- After the selection, use the toolbar or right-click the region:
- 框选完成后可以用工具条，也可以在选区上右键：
  - Copy to the clipboard / 复制到剪贴板
  - Pin to the desktop / 固定到桌面
  - Save to a folder (default `Pictures\Momontry` or `图片\Momontry`) / 保存到指定目录
  - Extract text with Windows OCR / 提取文字
- A pinned image can be dragged, resized with the wheel, closed with a double-click, and offers copy, save, text, and close from the right-click menu.
- 固定在桌面上的图片可以拖动，滚轮缩放，双击关闭，右键复制、保存、识别文字或关闭。

OCR uses the Windows language packs. Install one under Settings → Time & language → Language & region.

文字识别依赖系统的 OCR 语言包。可在“设置 → 时间和语言 → 语言和区域”中为当前语言安装“语言包 / 光学字符识别”。

Settings are stored in `%AppData%\momontry\config.json` (`lang` is `zh` or `en`).

配置文件在 `%AppData%\momontry\config.json`，其中 `lang` 为 `zh` 或 `en`。

## Build / 编译

Go 1.22 or newer. From the project directory:

需要 Go 1.22 或更高版本。在项目目录执行：

```powershell
.\build.ps1
```

`momontry.exe` has no console window. Download a built copy from the GitHub Releases page.

生成的 `momontry.exe` 没有控制台窗口。也可以从 GitHub Releases 下载已构建的程序。
