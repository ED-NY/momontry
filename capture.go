//go:build windows

package main

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"unsafe"

	"github.com/kbinani/screenshot"
	"github.com/lxn/win"
)

type captureSession struct {
	hwnd             win.HWND
	originX, originY int
	imgW, imgH       int
	img              *image.RGBA
	srcDC, dimDC     win.HDC
	srcBmp, dimBmp   win.HBITMAP
	srcOld, dimOld   win.HGDIOBJ
	released         bool
	dragging         bool
	hasSel           bool
	x0, y0, x1, y1   int
	prev             win.RECT
	hasPrev          bool
}

var session *captureSession

type toolButton struct {
	label  string
	action int
	rect   win.RECT
}

type toolbar struct {
	hwnd    win.HWND
	buttons []toolButton
	hover   int
	tracked bool
	w, h    int32
}

var bar *toolbar

func virtualBounds() image.Rectangle {
	// Do not call EnumDisplayMonitors here. Its Go callback can deadlock inside a window procedure.
	// 不用 EnumDisplayMonitors。它会在窗口回调里再进入 Go 回调，可能导致截屏线程卡死。
	x := int(win.GetSystemMetrics(win.SM_XVIRTUALSCREEN))
	y := int(win.GetSystemMetrics(win.SM_YVIRTUALSCREEN))
	w := int(win.GetSystemMetrics(win.SM_CXVIRTUALSCREEN))
	h := int(win.GetSystemMetrics(win.SM_CYVIRTUALSCREEN))
	if w <= 0 || h <= 0 {
		return image.Rectangle{}
	}
	return image.Rect(x, y, x+w, y+h)
}

func beginCapture() {
	if session != nil || hotDlg != 0 {
		return
	}
	bounds := virtualBounds()
	if bounds.Empty() {
		alert(T("noDisplay"))
		return
	}
	img, err := screenshot.CaptureRect(bounds)
	if err != nil || img == nil {
		if err == nil {
			err = errors.New(T("captureErr"))
		}
		logf("capture: %v", err)
		alert(tf("captureFail", err.Error()))
		return
	}
	s := &captureSession{
		originX: bounds.Min.X,
		originY: bounds.Min.Y,
		imgW:    img.Bounds().Dx(),
		imgH:    img.Bounds().Dy(),
		img:     img,
	}
	if err := s.buildBitmaps(); err != nil {
		logf("bitmap: %v", err)
		alert(T("bitmapFail"))
		return
	}
	session = s
	hwnd := win.CreateWindowEx(
		win.WS_EX_TOPMOST|win.WS_EX_TOOLWINDOW,
		classOverlay.ptr,
		titleApp.ptr,
		win.WS_POPUP,
		int32(s.originX), int32(s.originY), int32(s.imgW), int32(s.imgH),
		0, 0, hinst, nil,
	)
	if hwnd == 0 {
		session = nil
		s.release()
		alert(T("overlayFail"))
		return
	}
	s.hwnd = hwnd
	win.ShowWindow(hwnd, win.SW_SHOW)
	win.UpdateWindow(hwnd)
	win.SetForegroundWindow(hwnd)
	win.SetFocus(hwnd)
}

func (s *captureSession) buildBitmaps() error {
	var err error
	s.srcDC, s.srcBmp, s.srcOld, err = dibFromImage(s.img, false)
	if err != nil {
		return err
	}
	s.dimDC, s.dimBmp, s.dimOld, err = dibFromImage(s.img, true)
	if err != nil {
		s.release()
		return err
	}
	return nil
}

func dibFromImage(img *image.RGBA, dim bool) (win.HDC, win.HBITMAP, win.HGDIOBJ, error) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	var hdr win.BITMAPINFOHEADER
	hdr.BiSize = uint32(unsafe.Sizeof(hdr))
	hdr.BiWidth = int32(w)
	hdr.BiHeight = -int32(h)
	hdr.BiPlanes = 1
	hdr.BiBitCount = 32
	hdr.BiCompression = win.BI_RGB

	screen := win.GetDC(0)
	if screen == 0 {
		return 0, 0, 0, errors.New("GetDC")
	}
	defer win.ReleaseDC(0, screen)
	hdc := win.CreateCompatibleDC(screen)
	if hdc == 0 {
		return 0, 0, 0, errors.New("CreateCompatibleDC")
	}
	var bits unsafe.Pointer
	bmp := win.CreateDIBSection(hdc, &hdr, win.DIB_RGB_COLORS, &bits, 0, 0)
	if bmp == 0 || bits == nil {
		win.DeleteDC(hdc)
		return 0, 0, 0, errors.New("CreateDIBSection")
	}
	old := win.SelectObject(hdc, win.HGDIOBJ(bmp))
	dst := unsafe.Slice((*byte)(bits), w*h*4)
	for y := 0; y < h; y++ {
		si := img.PixOffset(b.Min.X, b.Min.Y+y)
		row := img.Pix[si : si+w*4]
		di := y * w * 4
		for x := 0; x < w; x++ {
			o := x * 4
			r := int(row[o])
			g := int(row[o+1])
			bl := int(row[o+2])
			if dim {
				// Use int. Multiplying a byte first overflows and turns bright pixels black.
				// 必须用 int。byte 先乘再除会溢出，亮色会被算成 0，框选前整屏变黑。
				r = r * 62 / 100
				g = g * 62 / 100
				bl = bl * 62 / 100
			}
			dst[di+o] = byte(bl)
			dst[di+o+1] = byte(g)
			dst[di+o+2] = byte(r)
			dst[di+o+3] = 255
		}
	}
	win.GdiFlush()
	return hdc, bmp, old, nil
}

func (s *captureSession) release() {
	if s == nil || s.released {
		return
	}
	s.released = true
	releaseDCBmp(s.srcDC, s.srcBmp, s.srcOld)
	releaseDCBmp(s.dimDC, s.dimBmp, s.dimOld)
	s.img = nil
}

func releaseDCBmp(dc win.HDC, bmp win.HBITMAP, old win.HGDIOBJ) {
	if dc != 0 && old != 0 {
		win.SelectObject(dc, old)
	}
	if bmp != 0 {
		win.DeleteObject(win.HGDIOBJ(bmp))
	}
	if dc != 0 {
		win.DeleteDC(dc)
	}
}

func invalidateOverlay(includeHint bool) {
	if session == nil || session.hwnd == 0 {
		return
	}
	x, y, w, h := session.sel()
	r := win.RECT{
		Left:   int32(x) - 8,
		Top:    int32(y) - 36,
		Right:  int32(x+w) + 8,
		Bottom: int32(y+h) + 8,
	}
	if session.hasPrev {
		if session.prev.Left < r.Left {
			r.Left = session.prev.Left
		}
		if session.prev.Top < r.Top {
			r.Top = session.prev.Top
		}
		if session.prev.Right > r.Right {
			r.Right = session.prev.Right
		}
		if session.prev.Bottom > r.Bottom {
			r.Bottom = session.prev.Bottom
		}
	}
	if includeHint {
		r.Left = 0
		if r.Top > 0 {
			r.Top = 0
		}
		if r.Right < int32(session.imgW) {
			r.Right = int32(session.imgW)
		}
		if r.Bottom < 80 {
			r.Bottom = 80
		}
	}
	cur := win.RECT{Left: int32(x) - 8, Top: int32(y) - 36, Right: int32(x+w) + 8, Bottom: int32(y+h) + 8}
	session.prev = cur
	session.hasPrev = true
	win.InvalidateRect(session.hwnd, &r, false)
}

func (s *captureSession) sel() (x, y, w, h int) {
	x0, x1 := s.x0, s.x1
	y0, y1 := s.y0, s.y1
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	return x0, y0, x1 - x0, y1 - y0
}

func (s *captureSession) selValid() bool {
	_, _, w, h := s.sel()
	return w >= 3 && h >= 3
}

func destroyCapture() {
	c := session
	b := bar
	session = nil
	bar = nil
	if b != nil && b.hwnd != 0 {
		win.DestroyWindow(b.hwnd)
	}
	if c != nil && c.hwnd != 0 {
		win.DestroyWindow(c.hwnd)
	}
	if c != nil {
		c.release()
	}
}

func finishCapture(action int) {
	if session == nil || !session.selValid() {
		if action == actCancel {
			destroyCapture()
		}
		return
	}
	x, y, w, h := session.sel()
	img := cropRGBA(session.img, image.Rect(x, y, x+w, y+h))
	screenX := int32(session.originX + x)
	screenY := int32(session.originY + y)
	destroyCapture()
	if img == nil {
		return
	}
	switch action {
	case actCopy:
		if err := copyImage(img); err != nil {
			alert(err.Error())
			return
		}
		toast(T("copied"))
	case actPin:
		openPin(img, screenX, screenY)
	case actSave:
		path, err := savePNG(img)
		if err != nil {
			alert(err.Error())
			return
		}
		toast(tf("saved", path))
	case actOCR:
		toast(T("ocrWorking"))
		go func() {
			text, err := recognize(img)
			invokeUI(func() {
				if err != nil {
					showTextWindow(T("ocrTitle"), tf("ocrFail", err.Error()))
					return
				}
				if text == "" {
					text = T("ocrEmpty")
				}
				showTextWindow(T("ocrTitle"), text)
			})
		}()
	default:
		// 取消
	}
}

func cropRGBA(src *image.RGBA, r image.Rectangle) *image.RGBA {
	if src == nil {
		return nil
	}
	r = r.Intersect(src.Bounds())
	if r.Empty() {
		return nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(dst, dst.Bounds(), src, r.Min, draw.Src)
	return dst
}

func overlayProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_ERASEBKGND:
		return 1
	case win.WM_PAINT:
		paintOverlay(hwnd)
		return 0
	case win.WM_LBUTTONDOWN:
		if session == nil {
			return 0
		}
		hideToolbar()
		session.dragging = true
		session.hasSel = true
		x := int(win.GET_X_LPARAM(lParam))
		y := int(win.GET_Y_LPARAM(lParam))
		session.x0, session.y0 = x, y
		session.x1, session.y1 = x, y
		win.SetCapture(hwnd)
		invalidateOverlay(true)
		return 0
	case win.WM_MOUSEMOVE:
		if session == nil || !session.dragging {
			return 0
		}
		session.x1 = int(win.GET_X_LPARAM(lParam))
		session.y1 = int(win.GET_Y_LPARAM(lParam))
		invalidateOverlay(false)
		return 0
	case win.WM_LBUTTONUP:
		if session == nil {
			return 0
		}
		if session.dragging {
			session.dragging = false
			win.ReleaseCapture()
			session.x1 = int(win.GET_X_LPARAM(lParam))
			session.y1 = int(win.GET_Y_LPARAM(lParam))
		}
		if session.selValid() {
			showToolbar()
		}
		invalidateOverlay(false)
		return 0
	case win.WM_RBUTTONUP:
		if session == nil {
			return 0
		}
		if !session.selValid() {
			destroyCapture()
			return 0
		}
		showToolbar()
		choice := popupChoice(hwnd)
		if choice != 0 {
			finishCapture(choice)
		}
		return 0
	case win.WM_KEYDOWN, win.WM_SYSKEYDOWN:
		if session == nil {
			return 0
		}
		switch wParam {
		case uintptr(win.VK_ESCAPE):
			destroyCapture()
		case uintptr(win.VK_RETURN):
			if session.selValid() {
				finishCapture(actCopy)
			}
		}
		return 0
	case win.WM_DESTROY:
		if session != nil && session.hwnd == hwnd {
			c := session
			b := bar
			session = nil
			bar = nil
			c.release()
			if b != nil && b.hwnd != 0 {
				win.DestroyWindow(b.hwnd)
			}
		}
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func paintOverlay(hwnd win.HWND) {
	if session == nil || session.dimDC == 0 {
		var ps win.PAINTSTRUCT
		hdc := win.BeginPaint(hwnd, &ps)
		win.EndPaint(hwnd, &ps)
		_ = hdc
		return
	}
	var ps win.PAINTSTRUCT
	hdc := win.BeginPaint(hwnd, &ps)
	win.BitBlt(hdc, 0, 0, int32(session.imgW), int32(session.imgH), session.dimDC, 0, 0, win.SRCCOPY)
	if session.hasSel {
		x, y, w, h := session.sel()
		if w > 0 && h > 0 && session.srcDC != 0 {
			win.BitBlt(hdc, int32(x), int32(y), int32(w), int32(h), session.srcDC, int32(x), int32(y), win.SRCCOPY)
			pen, _, _ := procCreatePen.Call(uintptr(win.PS_SOLID), 2, uintptr(win.RGB(0, 196, 168)))
			oldPen := win.SelectObject(hdc, win.HGDIOBJ(pen))
			oldBrush := win.SelectObject(hdc, win.GetStockObject(win.NULL_BRUSH))
			procRectangle.Call(uintptr(hdc), uintptr(int32(x)), uintptr(int32(y)), uintptr(int32(x+w)), uintptr(int32(y+h)))
			win.SelectObject(hdc, oldPen)
			win.SelectObject(hdc, oldBrush)
			if pen != 0 {
				win.DeleteObject(win.HGDIOBJ(pen))
			}
			drawSizeLabel(hdc, x, y, w, h)
		}
	} else {
		drawHint(hdc)
	}
	win.EndPaint(hwnd, &ps)
}

func drawHint(hdc win.HDC) {
	text := T("hint")
	buf, _ := utf16Slice(text)
	rect := win.RECT{Left: 0, Top: 24, Right: int32(session.imgW), Bottom: 64}
	old := win.SelectObject(hdc, win.HGDIOBJ(fontUI))
	win.SetBkMode(hdc, win.TRANSPARENT)
	shadow := rect
	shadow.Left++
	shadow.Top++
	win.SetTextColor(hdc, win.RGB(0, 0, 0))
	win.DrawTextEx(hdc, &buf[0], -1, &shadow, win.DT_CENTER|win.DT_SINGLELINE|win.DT_VCENTER, nil)
	win.SetTextColor(hdc, win.RGB(255, 255, 255))
	win.DrawTextEx(hdc, &buf[0], -1, &rect, win.DT_CENTER|win.DT_SINGLELINE|win.DT_VCENTER, nil)
	win.SelectObject(hdc, old)
}

func drawSizeLabel(hdc win.HDC, x, y, w, h int) {
	label := fmt.Sprintf("%d × %d", w, h)
	buf, _ := utf16Slice(label)
	top := int32(y) - 26
	if top < 4 {
		top = int32(y + h + 6)
	}
	rect := win.RECT{Left: int32(x), Top: top, Right: int32(x + 160), Bottom: top + 22}
	bg := rect
	bg.Left -= 6
	bg.Right = bg.Left + 150
	fillRect(hdc, &bg, win.RGB(0, 140, 120))
	old := win.SelectObject(hdc, win.HGDIOBJ(fontUI))
	win.SetBkMode(hdc, win.TRANSPARENT)
	win.SetTextColor(hdc, win.RGB(255, 255, 255))
	win.DrawTextEx(hdc, &buf[0], -1, &rect, win.DT_LEFT|win.DT_SINGLELINE|win.DT_VCENTER, nil)
	win.SelectObject(hdc, old)
}

func toolbarLabels() []struct {
	text   string
	action int
	w      int32
} {
	copyW, pinW, saveW, ocrW, cancelW := int32(72), int32(108), int32(72), int32(96), int32(72)
	if currentLang() == "en" {
		pinW, ocrW, cancelW = 64, 72, 84
	}
	return []struct {
		text   string
		action int
		w      int32
	}{
		{T("btnCopy"), actCopy, copyW},
		{T("btnPin"), actPin, pinW},
		{T("btnSave"), actSave, saveW},
		{T("btnOCR"), actOCR, ocrW},
		{T("btnCancel"), actCancel, cancelW},
	}
}

func showToolbar() {
	if session == nil || !session.selValid() {
		return
	}
	if bar != nil {
		placeToolbar()
		win.ShowWindow(bar.hwnd, win.SW_SHOW)
		win.InvalidateRect(bar.hwnd, nil, false)
		return
	}
	labels := toolbarLabels()
	var buttons []toolButton
	x := int32(8)
	for _, lb := range labels {
		buttons = append(buttons, toolButton{
			label:  lb.text,
			action: lb.action,
			rect:   win.RECT{Left: x, Top: 6, Right: x + lb.w, Bottom: 38},
		})
		x += lb.w + 6
	}
	width := x + 2
	height := int32(44)
	tb := &toolbar{buttons: buttons, hover: -1, w: width, h: height}
	bar = tb
	hwnd := win.CreateWindowEx(
		win.WS_EX_TOPMOST|win.WS_EX_TOOLWINDOW|wsExNoActivate,
		classToolbar.ptr,
		titleApp.ptr,
		win.WS_POPUP,
		0, 0, width, height,
		0, 0, hinst, nil,
	)
	tb.hwnd = hwnd
	placeToolbar()
	win.ShowWindow(hwnd, win.SW_SHOW)
	win.SetWindowPos(hwnd, win.HWND_TOPMOST, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|swpNoActivate)
}

func hideToolbar() {
	if bar != nil && bar.hwnd != 0 {
		win.ShowWindow(bar.hwnd, win.SW_HIDE)
	}
}

func placeToolbar() {
	if bar == nil || session == nil || bar.hwnd == 0 {
		return
	}
	x, y, w, h := session.sel()
	sx := int32(session.originX+x+w) - bar.w
	sy := int32(session.originY + y + h + 8)
	minX := int32(session.originX)
	minY := int32(session.originY)
	maxX := int32(session.originX + session.imgW)
	maxY := int32(session.originY + session.imgH)
	if sx < minX {
		sx = minX
	}
	if sx+bar.w > maxX {
		sx = maxX - bar.w
	}
	if sy+bar.h > maxY {
		sy = int32(session.originY+y) - bar.h - 8
	}
	if sy < minY {
		sy = minY
	}
	win.SetWindowPos(bar.hwnd, win.HWND_TOPMOST, sx, sy, bar.w, bar.h, swpNoActivate)
}

func toolbarProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_ERASEBKGND:
		return 1
	case win.WM_PAINT:
		paintToolbar(hwnd)
		return 0
	case win.WM_MOUSEMOVE:
		if bar == nil {
			return 0
		}
		if !bar.tracked {
			var ev win.TRACKMOUSEEVENT
			ev.CbSize = uint32(unsafe.Sizeof(ev))
			ev.DwFlags = 0x00000002 // TME_LEAVE
			ev.HwndTrack = hwnd
			win.TrackMouseEvent(&ev)
			bar.tracked = true
		}
		x := win.GET_X_LPARAM(lParam)
		y := win.GET_Y_LPARAM(lParam)
		hover := hitButton(x, y)
		if hover != bar.hover {
			bar.hover = hover
			win.InvalidateRect(hwnd, nil, false)
		}
		return 0
	case win.WM_MOUSELEAVE:
		if bar != nil {
			bar.tracked = false
			bar.hover = -1
			win.InvalidateRect(hwnd, nil, false)
		}
		return 0
	case win.WM_LBUTTONUP:
		if bar == nil {
			return 0
		}
		x := win.GET_X_LPARAM(lParam)
		y := win.GET_Y_LPARAM(lParam)
		if id := hitButton(x, y); id >= 0 {
			finishCapture(bar.buttons[id].action)
		}
		return 0
	case win.WM_KEYDOWN:
		if wParam == uintptr(win.VK_ESCAPE) {
			destroyCapture()
		}
		return 0
	}
	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func hitButton(x, y int32) int {
	if bar == nil {
		return -1
	}
	pt := win.POINT{X: x, Y: y}
	for i, b := range bar.buttons {
		if pt.X >= b.rect.Left && pt.X < b.rect.Right && pt.Y >= b.rect.Top && pt.Y < b.rect.Bottom {
			return i
		}
	}
	return -1
}

func paintToolbar(hwnd win.HWND) {
	var ps win.PAINTSTRUCT
	hdc := win.BeginPaint(hwnd, &ps)
	if bar == nil {
		win.EndPaint(hwnd, &ps)
		return
	}
	var rc win.RECT
	win.GetClientRect(hwnd, &rc)
	fillRect(hdc, &rc, win.RGB(28, 28, 28))
	pen, _, _ := procCreatePen.Call(uintptr(win.PS_SOLID), 1, uintptr(win.RGB(0, 168, 148)))
	oldPen := win.SelectObject(hdc, win.HGDIOBJ(pen))
	oldBrush := win.SelectObject(hdc, win.GetStockObject(win.NULL_BRUSH))
	procRectangle.Call(uintptr(hdc), 0, 0, uintptr(rc.Right-1), uintptr(rc.Bottom-1))
	win.SelectObject(hdc, oldPen)
	win.SelectObject(hdc, oldBrush)
	if pen != 0 {
		win.DeleteObject(win.HGDIOBJ(pen))
	}
	oldFont := win.SelectObject(hdc, win.HGDIOBJ(fontUI))
	win.SetBkMode(hdc, win.TRANSPARENT)
	for i, b := range bar.buttons {
		bg := b.rect
		if i == bar.hover {
			fillRect(hdc, &bg, win.RGB(0, 130, 114))
		} else {
			fillRect(hdc, &bg, win.RGB(48, 48, 48))
		}
		buf, _ := utf16Slice(b.label)
		win.SetTextColor(hdc, win.RGB(255, 255, 255))
		r := b.rect
		win.DrawTextEx(hdc, &buf[0], -1, &r, win.DT_CENTER|win.DT_VCENTER|win.DT_SINGLELINE, nil)
	}
	win.SelectObject(hdc, oldFont)
	win.EndPaint(hwnd, &ps)
}

func popupChoice(hwnd win.HWND) int {
	menu := win.CreatePopupMenu()
	if menu == 0 {
		return 0
	}
	defer win.DestroyMenu(menu)
	appendMenuItem(menu, win.MF_STRING, actCopy, T("menuCopy"))
	appendMenuItem(menu, win.MF_STRING, actPin, T("menuPin"))
	appendMenuItem(menu, win.MF_STRING, actSave, T("menuSave"))
	appendMenuItem(menu, win.MF_STRING, actOCR, T("menuOCR"))
	appendMenuItem(menu, win.MF_SEPARATOR, 0, "")
	appendMenuItem(menu, win.MF_STRING, actCancel, T("btnCancel"))
	var pt win.POINT
	win.GetCursorPos(&pt)
	win.SetForegroundWindow(hwnd)
	cmd := win.TrackPopupMenu(menu, win.TPM_RETURNCMD|win.TPM_NONOTIFY, pt.X, pt.Y, 0, hwnd, nil)
	win.PostMessage(hwnd, wmNull, 0, 0)
	return int(cmd)
}
