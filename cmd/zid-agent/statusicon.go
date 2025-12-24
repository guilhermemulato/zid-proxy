package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"sync"

	"fyne.io/fyne/v2"
	"github.com/guilherme/zid-proxy/internal/agentui"
)

var (
	iconOnce    sync.Once
	iconUnknown []byte
	iconOK      []byte
	iconFail    []byte
)

func statusIconResource(status agentui.HeartbeatStatus) fyne.Resource {
	iconOnce.Do(func() {
		iconUnknown = mustMakeDotPNG(color.RGBA{R: 150, G: 150, B: 150, A: 255})
		iconOK = mustMakeDotPNG(color.RGBA{R: 0, G: 180, B: 0, A: 255})
		iconFail = mustMakeDotPNG(color.RGBA{R: 200, G: 0, B: 0, A: 255})
	})

	var data []byte
	switch status.State {
	case agentui.HeartbeatOK:
		data = iconOK
	case agentui.HeartbeatFail:
		data = iconFail
	default:
		data = iconUnknown
	}

	return fyne.NewStaticResource("zid-agent-status.png", data)
}

func mustMakeDotPNG(dot color.RGBA) []byte {
	const size = 16

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	center := float64(size-1) / 2
	radius := center - 1
	r2 := radius * radius

	border := color.RGBA{R: 20, G: 20, B: 20, A: 255}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - center
			dy := float64(y) - center
			dist2 := dx*dx + dy*dy
			if dist2 <= r2 {
				img.SetRGBA(x, y, dot)
			}
			if dist2 <= r2 && dist2 >= (r2-2.5) {
				img.SetRGBA(x, y, border)
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}
