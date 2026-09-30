//go:build windows

package main

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const ocrScript = `
param(
  [Parameter(Mandatory = $true)][string]$ImagePath,
  [Parameter(Mandatory = $true)][string]$OutPath
)
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$null = [Windows.Storage.StorageFile, Windows.Storage, ContentType = WindowsRuntime]
$null = [Windows.Media.Ocr.OcrEngine, Windows.Foundation, ContentType = WindowsRuntime]
$null = [Windows.Media.Ocr.OcrResult, Windows.Foundation, ContentType = WindowsRuntime]
$null = [Windows.Graphics.Imaging.BitmapDecoder, Windows.Graphics.Imaging, ContentType = WindowsRuntime]
$null = [Windows.Graphics.Imaging.SoftwareBitmap, Windows.Graphics.Imaging, ContentType = WindowsRuntime]
$null = [Windows.Globalization.Language, Windows.Globalization, ContentType = WindowsRuntime]
$null = [Windows.Storage.Streams.IRandomAccessStream, Windows.Storage.Streams, ContentType = WindowsRuntime]

$asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object {
  $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation` + "`" + `1'
})[0]

function Await([object]$WinRtTask, [type]$ResultType) {
  $asTask = $asTaskGeneric.MakeGenericMethod($ResultType)
  $netTask = $asTask.Invoke($null, @($WinRtTask))
  $netTask.Wait(-1) | Out-Null
  $netTask.Result
}

$file = Await ([Windows.Storage.StorageFile]::GetFileFromPathAsync($ImagePath)) ([Windows.Storage.StorageFile])
$stream = Await ($file.OpenAsync([Windows.Storage.FileAccessMode]::Read)) ([Windows.Storage.Streams.IRandomAccessStream])
$decoder = Await ([Windows.Graphics.Imaging.BitmapDecoder]::CreateAsync($stream)) ([Windows.Graphics.Imaging.BitmapDecoder])
$bitmap = $null
try {
  $bitmap = Await ($decoder.GetSoftwareBitmapAsync([Windows.Graphics.Imaging.BitmapPixelFormat]::Bgra8, [Windows.Graphics.Imaging.BitmapAlphaMode]::Premultiplied)) ([Windows.Graphics.Imaging.SoftwareBitmap])
} catch {
  $bitmap = Await ($decoder.GetSoftwareBitmapAsync()) ([Windows.Graphics.Imaging.SoftwareBitmap])
}
$engine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromUserProfileLanguages()
if (-not $engine) {
  foreach ($tag in @('zh-Hans', 'zh-Hant', 'en-US', 'en')) {
    try {
      $lang = New-Object Windows.Globalization.Language $tag
      $engine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromLanguage($lang)
      if ($engine) { break }
    } catch {}
  }
}
if (-not $engine) { throw 'OCR_PACK_MISSING' }
$result = Await ($engine.RecognizeAsync($bitmap)) ([Windows.Media.Ocr.OcrResult])
$lines = New-Object System.Collections.Generic.List[string]
foreach ($line in $result.Lines) { $lines.Add([string]$line.Text) }
$text = if ($lines.Count -gt 0) { $lines -join "` + "`r`n" + `" } else { [string]$result.Text }
[System.IO.File]::WriteAllText($OutPath, $text, (New-Object System.Text.UTF8Encoding $false))
`

func recognize(img *image.RGBA) (string, error) {
	if img == nil {
		return "", fmt.Errorf("%s", T("noImage"))
	}
	dir, err := os.MkdirTemp("", "momontry-ocr")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	imgPath := filepath.Join(dir, "shot.png")
	outPath := filepath.Join(dir, "out.txt")
	scriptPath := filepath.Join(dir, "ocr.ps1")
	f, err := os.Create(imgPath)
	if err != nil {
		return "", err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return "", err
	}
	f.Close()
	if err := os.WriteFile(scriptPath, append([]byte{0xEF, 0xBB, 0xBF}, []byte(ocrScript)...), 0o644); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe",
		"-NoProfile", "-NonInteractive", "-STA", "-WindowStyle", "Hidden",
		"-ExecutionPolicy", "Bypass", "-File", scriptPath, imgPath, outPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("%s", T("ocrTimeout"))
	}
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		if strings.Contains(msg, "OCR_PACK_MISSING") {
			return "", fmt.Errorf("%s", T("ocrPackMissing"))
		}
		return "", fmt.Errorf("%s", msg)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		if len(output) > 0 {
			return "", fmt.Errorf("%s", output)
		}
		return "", err
	}
	return string(data), nil
}
