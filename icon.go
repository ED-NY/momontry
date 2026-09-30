//go:build windows

package main

import (
	"bytes"
	"encoding/binary"
)

func trayIcon() []byte {
	images := []struct {
		size int
		pix  []byte
	}{
		{16, drawIcon(16)},
		{32, drawIcon(32)},
	}
	return encodeICO(images)
}

func drawIcon(n int) []byte {
	pix := make([]byte, n*n*4)
	set := func(x, y int, r, g, b, a byte) {
		if x < 0 || y < 0 || x >= n || y >= n {
			return
		}
		i := (y*n + x) * 4
		pix[i] = b
		pix[i+1] = g
		pix[i+2] = r
		pix[i+3] = a
	}
	margin := n / 16
	if margin < 1 {
		margin = 1
	}
	rad := n / 5
	if rad < 2 {
		rad = 2
	}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if inRounded(x, y, margin, margin, n-1-margin, n-1-margin, rad) {
				set(x, y, 20, 143, 134, 255)
			}
		}
	}
	thick := n / 16
	if thick < 1 {
		thick = 1
	}
	length := n / 3
	if length < 3 {
		length = 3
	}
	inset := n / 5
	if inset < 2 {
		inset = 2
	}
	corners := [][2]int{
		{inset, inset},
		{n - 1 - inset, inset},
		{inset, n - 1 - inset},
		{n - 1 - inset, n - 1 - inset},
	}
	dirs := [][2]int{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}}
	for i, c := range corners {
		dx, dy := dirs[i][0], dirs[i][1]
		for k := 0; k < length; k++ {
			for t := 0; t < thick; t++ {
				set(c[0]+dx*k, c[1]+dy*t, 255, 255, 255, 255)
				set(c[0]+dx*t, c[1]+dy*k, 255, 255, 255, 255)
			}
		}
	}
	return pix
}

func inRounded(x, y, l, t, r, b, rad int) bool {
	if x < l || y < t || x > r || y > b {
		return false
	}
	type corner struct{ cx, cy int }
	corners := []corner{{l + rad, t + rad}, {r - rad, t + rad}, {l + rad, b - rad}, {r - rad, b - rad}}
	for _, c := range corners {
		inX := (c.cx == l+rad && x < l+rad) || (c.cx == r-rad && x > r-rad)
		inY := (c.cy == t+rad && y < t+rad) || (c.cy == b-rad && y > b-rad)
		if inX && inY {
			dx := x - c.cx
			dy := y - c.cy
			if dx*dx+dy*dy > rad*rad {
				return false
			}
		}
	}
	return true
}

func encodeICO(images []struct {
	size int
	pix  []byte
}) []byte {
	count := len(images)
	offset := 6 + 16*count
	var blobs [][]byte
	for _, im := range images {
		blobs = append(blobs, dibIcon(im.size, im.pix))
	}
	buf := bytes.NewBuffer(make([]byte, 0, offset+4096))
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(count))
	for i, im := range images {
		sz := byte(im.size)
		if im.size >= 256 {
			sz = 0
		}
		buf.WriteByte(sz)
		buf.WriteByte(sz)
		buf.WriteByte(0)
		buf.WriteByte(0)
		_ = binary.Write(buf, binary.LittleEndian, uint16(1))
		_ = binary.Write(buf, binary.LittleEndian, uint16(32))
		_ = binary.Write(buf, binary.LittleEndian, uint32(len(blobs[i])))
		_ = binary.Write(buf, binary.LittleEndian, uint32(offset))
		offset += len(blobs[i])
	}
	for _, b := range blobs {
		buf.Write(b)
	}
	return buf.Bytes()
}

func dibIcon(size int, topDown []byte) []byte {
	xorStride := size * 4
	xorSize := xorStride * size
	maskStride := ((size + 31) / 32) * 4
	maskSize := maskStride * size
	buf := bytes.NewBuffer(make([]byte, 0, 40+xorSize+maskSize))
	_ = binary.Write(buf, binary.LittleEndian, uint32(40))
	_ = binary.Write(buf, binary.LittleEndian, int32(size))
	_ = binary.Write(buf, binary.LittleEndian, int32(size*2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(32))
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))
	_ = binary.Write(buf, binary.LittleEndian, uint32(xorSize+maskSize))
	_ = binary.Write(buf, binary.LittleEndian, int32(0))
	_ = binary.Write(buf, binary.LittleEndian, int32(0))
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))
	row := make([]byte, xorStride)
	for y := size - 1; y >= 0; y-- {
		copy(row, topDown[y*xorStride:(y+1)*xorStride])
		buf.Write(row)
	}
	buf.Write(make([]byte, maskSize))
	return buf.Bytes()
}
