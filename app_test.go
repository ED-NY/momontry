//go:build windows

package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/lxn/win"
)

func TestFormatHotkey(t *testing.T) {
	got := formatHotkey(modControl|modAlt, 0x41)
	if got != "Ctrl+Alt+A" {
		t.Fatalf("got %q", got)
	}
	if formatHotkey(modShift, 0) != "Shift" {
		t.Fatalf("modifier only: %q", formatHotkey(modShift, 0))
	}
	if keyName(vkF1) != "F1" {
		t.Fatalf("F1")
	}
	if !isModifierVK(vkLControl) || allowsBareKey('A') {
		t.Fatal("modifier classification")
	}
	if !allowsBareKey(vkF1) {
		t.Fatal("function key should be allowed alone")
	}
}

func TestDIBRoundTrip(t *testing.T) {
	runtime.LockOSThread()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 220, G: 30, B: 40, A: 255})
		}
	}
	dc, bmp, old, err := dibFromImage(img, false)
	if err != nil {
		t.Fatal(err)
	}
	defer releaseDCBmp(dc, bmp, old)
	px := win.GetPixel(dc, 3, 3)
	if px != win.RGB(220, 30, 40) {
		t.Fatalf("dib pixel = %#x, want red", px)
	}
	dimDC, dimBmp, dimOld, err := dibFromImage(img, true)
	if err != nil {
		t.Fatal(err)
	}
	defer releaseDCBmp(dimDC, dimBmp, dimOld)
	dimmed := win.GetPixel(dimDC, 3, 3)
	if dimmed != win.RGB(136, 18, 24) {
		t.Fatalf("dimmed pixel = %#x, want scaled red", dimmed)
	}
}

func TestTrayIcon(t *testing.T) {
	ico := trayIcon()
	if len(ico) < 22 || ico[0] != 0 || ico[1] != 0 || ico[2] != 1 || ico[3] != 0 {
		t.Fatalf("bad ico header %v", ico[:6])
	}
}

func TestNormalizeConfig(t *testing.T) {
	c := Config{}
	normalizeConfig(&c)
	if c.Key != 0x41 || c.Modifiers != (modControl|modAlt) || c.SaveDir == "" || c.Lang != "zh" {
		t.Fatalf("%+v", c)
	}
}

func TestI18N(t *testing.T) {
	if zh["shot"] == en["shot"] || en["shot"] != "Capture" || zh["shot"] != "截屏" {
		t.Fatalf("tables %q %q", zh["shot"], en["shot"])
	}
	for key := range zh {
		if _, ok := en[key]; !ok {
			t.Fatalf("missing en key %s", key)
		}
	}
	for key := range en {
		if _, ok := zh[key]; !ok {
			t.Fatalf("missing zh key %s", key)
		}
	}
}

func TestRecognize(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "in.png")
	ps := "$ErrorActionPreference='Stop'; Add-Type -AssemblyName System.Drawing; $bmp = New-Object System.Drawing.Bitmap 720,180; $g = [System.Drawing.Graphics]::FromImage($bmp); $g.Clear([System.Drawing.Color]::White); $g.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::AntiAlias; $font = New-Object System.Drawing.Font 'Arial', 64; $g.DrawString('HELLO', $font, [System.Drawing.Brushes]::Black, 30, 40); $bmp.Save('" + pngPath + "', [System.Drawing.Imaging.ImageFormat]::Png); $g.Dispose(); $bmp.Dispose()"
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", ps)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("draw image: %v %s", err, out)
	}
	f, err := os.Open(pngPath)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	b := decoded.Bounds()
	img := image.NewRGBA(b)
	draw.Draw(img, b, decoded, b.Min, draw.Src)
	text, err := recognize(img)
	if err != nil {
		if strings.Contains(err.Error(), "OCR_PACK_MISSING") || strings.Contains(err.Error(), "语言包") || strings.Contains(err.Error(), "language pack") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToUpper(text), "HELLO") {
		t.Fatalf("ocr text %q", text)
	}
}
