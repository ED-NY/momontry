//go:build resources

package main

import (
	"image"
	"os"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
)

func main() {
	sizes := []int{16, 24, 32, 48, 64, 128, 256}
	images := make([]image.Image, 0, len(sizes))
	for _, n := range sizes {
		images = append(images, iconNRGBA(n))
	}
	icon, err := winres.NewIconFromImages(images)
	if err != nil {
		panic(err)
	}
	var rs winres.ResourceSet
	if err := rs.SetIcon(winres.ID(1), icon); err != nil {
		panic(err)
	}
	rs.SetManifest(winres.AppManifest{
		Description:         "Momontry 截图",
		DPIAwareness:        winres.DPIPerMonitorV2,
		UseCommonControlsV6: true,
	})
	var vi version.Info
	vi.SetFileVersion("1.0.0")
	vi.SetProductVersion("1.0.0")
	_ = vi.Set(0x0804, "ProductName", "Momontry")
	_ = vi.Set(0x0804, "FileDescription", "Momontry 截图")
	_ = vi.Set(0x0804, "OriginalFilename", "momontry.exe")
	rs.SetVersionInfo(vi)

	out, err := os.Create("rsrc_windows_amd64.syso")
	if err != nil {
		panic(err)
	}
	defer out.Close()
	if err := rs.WriteObject(out, winres.ArchAMD64); err != nil {
		panic(err)
	}
}

func iconNRGBA(n int) image.Image {
	pix := drawIcon(n)
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for i := 0; i < len(pix); i += 4 {
		img.Pix[i] = pix[i+2]
		img.Pix[i+1] = pix[i+1]
		img.Pix[i+2] = pix[i]
		img.Pix[i+3] = pix[i+3]
	}
	return img
}
