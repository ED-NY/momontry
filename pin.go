//go:build windows

package main

import (
	"fmt"
	"image"
	"unsafe"

	"github.com/lxn/win"
)

type pinWindow struct {
	hwnd       win.HWND
	img        *image.RGBA
	imgW       int32
	imgH       int32
	scale      float64
	dc         win.HDC
	bmp        win.HBITMAP
	old        win.HGDIOBJ
	dragging   bool
	dragX      int32
	dragY      int32
	originLeft int32
	originTop  int32
}

var (
	pins         []*pinWindow
	pinHintShown bool
)

func openPin(img *image.RGBA, screenX, screenY int32) {
	if img == nil {
		return
	}
	w := int32(img.Bounds().Dx())
	h := int32(img.Bounds().Dy())
	if w < 1 || h < 1 {
		return
	}
	dc, bmp, old, err := dibFromImage(img, false)
	if err != nil {
		alert(T("pinFail"))
		return
	}
	scale := 1.0
	wa := workArea()
	maxW := int32(float64(wa.Right-wa.Left) * 0.9)
	maxH := int32(float64(wa.Bottom-wa.Top) * 0.9)
	if w > maxW {
		scale = float64(maxW) / float64(w)
	}
	if float64(h)*scale > float64(maxH) {
		scale = float64(maxH) / float64(h)
	}
	p := &pinWindow{
		img:   img,
		imgW:  w,
		imgH:  h,
		scale: scale,
		dc:    dc,
		bmp:   bmp,
		old:   old,
	}
	pins = append(pins, p)
	pw := int32(float64(w) * scale)
	ph := int32(float64(h) * scale)
	if pw < 16 {
		pw = 16
	}
	if ph < 16 {
		ph = 16
	}
	hwnd := win.CreateWindowEx(
		win.WS_EX_TOPMOST|win.WS_EX_TOOLWINDOW,
		classPin.ptr,
		titleApp.ptr,
		win.WS_POPUP|win.WS_VISIBLE,
		screenX, screenY, pw, ph,
		0, 0, hinst, unsafe.Pointer(p),
	)
	if hwnd == 0 {
		p.release()
		removePin(p)
		alert(T("pinWindowFail"))
		return
	}
	p.hwnd = hwnd
	win.SetWindowPos(hwnd, win.HWND_TOPMOST, screenX, screenY, pw, ph, swpNoActivate)
	if !pinHintShown {
		pinHintShown = true
		toast(T("pinnedHint"))
	}
}

func removePin(p *pinWindow) {
	out := pins[:0]
	for _, it := range pins {
		if it != p {
			out = append(out, it)
		}
	}
	pins = out
}

func (p *pinWindow) release() {
	if p == nil {
		return
	}
	releaseDCBmp(p.dc, p.bmp, p.old)
	p.dc = 0
	p.bmp = 0
	p.img = nil
}

func pinByHWND(hwnd win.HWND) *pinWindow {
	if v := getUserData(hwnd); v != 0 {
		return (*pinWindow)(unsafe.Pointer(v))
	}
	return nil
}

func pinProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == win.WM_NCCREATE {
		cs := (*win.CREATESTRUCT)(unsafe.Pointer(lParam))
		setUserData(hwnd, cs.CreateParams)
		return 1
	}
	p := pinByHWND(hwnd)
	if p == nil {
		return win.DefWindowProc(hwnd, msg, wParam, lParam)
	}
	switch msg {
	case win.WM_ERASEBKGND:
		return 1
	case win.WM_PAINT:
		paintPin(hwnd, p)
		return 0
	case win.WM_LBUTTONDOWN:
		p.dragging = true
		var pt win.POINT
		win.GetCursorPos(&pt)
		var wr win.RECT
		win.GetWindowRect(hwnd, &wr)
		p.dragX = pt.X
		p.dragY = pt.Y
		p.originLeft = wr.Left
		p.originTop = wr.Top
		win.SetCapture(hwnd)
		return 0
	case win.WM_MOUSEMOVE:
		if !p.dragging {
			return 0
		}
		var pt win.POINT
		win.GetCursorPos(&pt)
		nx := p.originLeft + (pt.X - p.dragX)
		ny := p.originTop + (pt.Y - p.dragY)
		var wr win.RECT
		win.GetWindowRect(hwnd, &wr)
		win.SetWindowPos(hwnd, win.HWND_TOPMOST, nx, ny, wr.Right-wr.Left, wr.Bottom-wr.Top, swpNoActivate)
		return 0
	case win.WM_LBUTTONUP:
		p.dragging = false
		win.ReleaseCapture()
		return 0
	case win.WM_LBUTTONDBLCLK:
		closePin(hwnd, p)
		return 0
	case win.WM_MOUSEWHEEL:
		delta := int16(wParam >> 16)
		if delta > 0 {
			p.scale *= 1.1
		} else if delta < 0 {
			p.scale /= 1.1
		}
		if p.scale < 0.15 {
			p.scale = 0.15
		}
		if p.scale > 6 {
			p.scale = 6
		}
		nw := int32(float64(p.imgW) * p.scale)
		nh := int32(float64(p.imgH) * p.scale)
		if nw < 16 {
			nw = 16
		}
		if nh < 16 {
			nh = 16
		}
		var wr win.RECT
		win.GetWindowRect(hwnd, &wr)
		cx := wr.Left + (wr.Right-wr.Left)/2
		cy := wr.Top + (wr.Bottom-wr.Top)/2
		win.SetWindowPos(hwnd, win.HWND_TOPMOST, cx-nw/2, cy-nh/2, nw, nh, swpNoActivate)
		win.InvalidateRect(hwnd, nil, false)
		return 0
	case win.WM_RBUTTONUP:
		switch pinMenu(hwnd) {
		case actCopy:
			if err := copyImage(p.img); err != nil {
				alert(err.Error())
			} else {
				toast(T("copied"))
			}
		case actSave:
			path, err := savePNG(p.img)
			if err != nil {
				alert(err.Error())
			} else {
				toast(tf("saved", path))
			}
		case actOCR:
			img := p.img
			toast(T("ocrWorking"))
			go func() {
				text, err := recognize(img)
				invokeUI(func() {
					if err != nil {
						showTextWindow(T("ocrTitle"), tf("ocrFail", err.Error()))
						return
					}
					if text == "" {
						text = T("ocrEmptyShort")
					}
					showTextWindow(T("ocrTitle"), text)
				})
			}()
		case actCancel:
			closePin(hwnd, p)
		}
		return 0
	case win.WM_DESTROY:
		p.release()
		removePin(p)
		setUserData(hwnd, 0)
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func closePin(hwnd win.HWND, p *pinWindow) {
	removePin(p)
	win.DestroyWindow(hwnd)
}

func paintPin(hwnd win.HWND, p *pinWindow) {
	var ps win.PAINTSTRUCT
	hdc := win.BeginPaint(hwnd, &ps)
	var rc win.RECT
	win.GetClientRect(hwnd, &rc)
	if p.dc != 0 {
		win.SetStretchBltMode(hdc, win.HALFTONE)
		win.StretchBlt(hdc, 0, 0, rc.Right, rc.Bottom, p.dc, 0, 0, p.imgW, p.imgH, win.SRCCOPY)
	}
	pen, _, _ := procCreatePen.Call(uintptr(win.PS_SOLID), 1, uintptr(win.RGB(0, 160, 140)))
	oldPen := win.SelectObject(hdc, win.HGDIOBJ(pen))
	oldBrush := win.SelectObject(hdc, win.GetStockObject(win.NULL_BRUSH))
	procRectangle.Call(uintptr(hdc), 0, 0, uintptr(rc.Right-1), uintptr(rc.Bottom-1))
	win.SelectObject(hdc, oldPen)
	win.SelectObject(hdc, oldBrush)
	if pen != 0 {
		win.DeleteObject(win.HGDIOBJ(pen))
	}
	win.EndPaint(hwnd, &ps)
}

func pinMenu(hwnd win.HWND) int {
	menu := win.CreatePopupMenu()
	if menu == 0 {
		return 0
	}
	defer win.DestroyMenu(menu)
	appendMenuItem(menu, win.MF_STRING, actCopy, T("menuCopy"))
	appendMenuItem(menu, win.MF_STRING, actSave, T("menuSave"))
	appendMenuItem(menu, win.MF_STRING, actOCR, T("menuOCR"))
	appendMenuItem(menu, win.MF_SEPARATOR, 0, "")
	appendMenuItem(menu, win.MF_STRING, actCancel, T("menuClose"))
	var pt win.POINT
	win.GetCursorPos(&pt)
	win.SetForegroundWindow(hwnd)
	cmd := win.TrackPopupMenu(menu, win.TPM_RETURNCMD|win.TPM_NONOTIFY, pt.X, pt.Y, 0, hwnd, nil)
	win.PostMessage(hwnd, wmNull, 0, 0)
	return int(cmd)
}

func savePNG(img *image.RGBA) (string, error) {
	if img == nil {
		return "", fmt.Errorf("%s", T("noImage"))
	}
	return writePNG(img)
}
