//go:build windows

package main

import (
	"errors"
	"image"
	"unsafe"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

func copyImage(img *image.RGBA) error {
	if img == nil {
		return errors.New(T("noImage"))
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return errors.New(T("emptyImage"))
	}
	stride := ((w*32 + 31) / 32) * 4
	total := 40 + stride*h
	hmem := win.GlobalAlloc(win.GMEM_MOVEABLE, uintptr(total))
	if hmem == 0 {
		return errors.New(T("clipAlloc"))
	}
	ptr := win.GlobalLock(hmem)
	if ptr == nil {
		win.GlobalFree(hmem)
		return errors.New(T("clipLock"))
	}
	hdr := (*win.BITMAPINFOHEADER)(ptr)
	*hdr = win.BITMAPINFOHEADER{
		BiSize:        40,
		BiWidth:       int32(w),
		BiHeight:      int32(h),
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: win.BI_RGB,
		BiSizeImage:   uint32(stride * h),
	}
	dst := unsafe.Slice((*byte)(unsafe.Add(ptr, 40)), stride*h)
	for y := 0; y < h; y++ {
		srcY := (h - 1 - y) + b.Min.Y
		di := y * stride
		for x := 0; x < w; x++ {
			si := img.PixOffset(b.Min.X+x, srcY)
			dst[di] = img.Pix[si+2]
			dst[di+1] = img.Pix[si+1]
			dst[di+2] = img.Pix[si]
			dst[di+3] = 255
			di += 4
		}
	}
	win.GlobalUnlock(hmem)
	if !win.OpenClipboard(msgHwnd) {
		win.GlobalFree(hmem)
		return errors.New(T("clipOpen"))
	}
	win.EmptyClipboard()
	if win.SetClipboardData(win.CF_DIB, win.HANDLE(hmem)) == 0 {
		win.CloseClipboard()
		win.GlobalFree(hmem)
		return errors.New(T("clipWrite"))
	}
	win.CloseClipboard()
	return nil
}

func copyText(s string) error {
	utf, err := windows.UTF16FromString(s)
	if err != nil {
		return err
	}
	size := uintptr(len(utf) * 2)
	hmem := win.GlobalAlloc(win.GMEM_MOVEABLE, size)
	if hmem == 0 {
		return errors.New(T("clipAlloc"))
	}
	ptr := win.GlobalLock(hmem)
	if ptr == nil {
		win.GlobalFree(hmem)
		return errors.New(T("clipLock"))
	}
	copy(unsafe.Slice((*uint16)(ptr), len(utf)), utf)
	win.GlobalUnlock(hmem)
	if !win.OpenClipboard(msgHwnd) {
		win.GlobalFree(hmem)
		return errors.New(T("clipOpen"))
	}
	win.EmptyClipboard()
	if win.SetClipboardData(cfUnicodeText, win.HANDLE(hmem)) == 0 {
		win.CloseClipboard()
		win.GlobalFree(hmem)
		return errors.New(T("clipWrite"))
	}
	win.CloseClipboard()
	return nil
}
