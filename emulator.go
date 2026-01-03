package emul8

import (
	"context"
	"emul8/chip8"
	"fmt"
	"image"
	"image/color"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var keyMap = map[fyne.KeyName]uint8{
	fyne.Key1: 0x1, fyne.Key2: 0x2, fyne.Key3: 0x3, fyne.Key4: 0xC,
	fyne.KeyQ: 0x4, fyne.KeyW: 0x5, fyne.KeyE: 0x6, fyne.KeyR: 0xD,
	fyne.KeyA: 0x7, fyne.KeyS: 0x8, fyne.KeyD: 0x9, fyne.KeyF: 0xE,
	fyne.KeyZ: 0xA, fyne.KeyX: 0x0, fyne.KeyC: 0xB, fyne.KeyV: 0xF,
}

type Emulator struct {
	beep    Beep
	running atomic.Bool
}

func (e *Emulator) onKeyDown(k *fyne.KeyEvent) {
	if hex, ok := keyMap[k.Name]; ok {
		chip8.SetKey(hex, true)
	}
}

func (e *Emulator) onKeyUp(k *fyne.KeyEvent) {
	if hex, ok := keyMap[k.Name]; ok {
		chip8.SetKey(hex, false)
	}
}

func (e *Emulator) Load(b []byte) {
	chip8.Reset()
	chip8.Load(b)
}

func (e *Emulator) Run() {
	a := app.New()
	w := a.NewWindow("Chip-8 Emulator")

	// Create a back-buffer for the pixel data
	buffer := image.NewRGBA(image.Rect(0, 0, chip8.Width, chip8.Height))

	image := canvas.NewImageFromImage(buffer)
	image.FillMode = canvas.ImageFillStretch  // Scales the grid to window size
	image.ScaleMode = canvas.ImageScalePixels // Maintains "pixelated" retro look

	canv, ok := w.Canvas().(desktop.Canvas) // Extension that exposes OnKeyUp event
	if !ok {
		panic("emulator cannot be run on mobile")
	}
	canv.SetOnKeyDown(e.onKeyDown)
	canv.SetOnKeyUp(e.onKeyUp)

	data := make([]string, 0)

	bound := binding.BindStringList(&data)

	list := widget.NewListWithData(
		bound,
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(di binding.DataItem, obj fyne.CanvasObject) {
			s, _ := di.(binding.String).Get()
			obj.(*widget.Label).SetText(s)
		},
	)

	var paused atomic.Bool
	var next atomic.Bool

	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.MediaPlayIcon(), func() {
			paused.Store(false)
		}),
		widget.NewToolbarAction(theme.MediaPauseIcon(), func() {
			paused.Store(true)
		}),
		widget.NewToolbarAction(theme.MediaSkipNextIcon(), func() {
			next.Store(true)
		}),
	)

	box := container.NewBorder(toolbar, nil, list, nil, image)

	w.SetContent(box)

	w.Resize(fyne.NewSize(float32(chip8.Width*10), float32(chip8.Height*10))) // 10x scale for visibility

	e.running.Store(true)

	var wg sync.WaitGroup

	wg.Go(func() {
		defer func() {
			_ = e.beep.Stop()
		}()

		cpuTicker := time.NewTicker(chip8.ClockRate)
		defer cpuTicker.Stop()

		var ctr int

		for range cpuTicker.C {
			if !e.running.Load() {
				break
			}

			if paused.Load() {
				if !next.Load() {
					continue
				}
				next.Store(false)
			}

			if ctr < 1 {
				opcode := chip8.Opcode(chip8.ProgramCounter())
				opstr := fmt.Sprintf("%04X", opcode)

				data = append([]string{opstr}, data...)
				ctr++
			}

			if chip8.ProgramCounter()+2 <= chip8.LastAddress {
				opcode := chip8.Opcode(chip8.ProgramCounter() + 2)
				opstr := fmt.Sprintf("%04X", opcode)

				data = append([]string{opstr}, data...)
			}

			if len(data) > 10 {
				data = data[:10]
			}

			_ = bound.Reload()

			info := chip8.Step()

			redraw := (info & chip8.Redraw) != 0
			sound := (info & chip8.Sound) != 0

			if sound {
				_ = e.beep.Start(context.Background())
			} else {
				_ = e.beep.Stop()
			}

			if redraw {
				for i, val := range chip8.Display() {
					x, y := i%chip8.Width, i/chip8.Width
					c := color.Black
					if val == 1 {
						c = color.White
					}
					buffer.Set(x, y, c) // Directly sets pixels in the buffer
				}

				fyne.Do(func() {
					image.Refresh()
				})
			}

			fyne.Do(func() {
				list.Select(widget.ListItemID(1))
				list.ScrollToTop()
				list.Refresh()
			})
		}
	})

	w.ShowAndRun()
	e.running.Store(false)
	wg.Wait()
}
