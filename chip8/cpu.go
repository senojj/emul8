/*
 * Copyright 2026 Joshua Jones <joshua.jones.software@gmail.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      www.apache.org
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package chip8

import (
	"sync/atomic"
	"time"
)

const (
	RegisterCount       int    = 16
	KeyCount            int    = 16
	FontStartAddress    uint16 = 0x50
	LastAddress         uint16 = 0xFFE
	ProgramStartAddress uint16 = 0x200
	CarryFlag           uint8  = 0xF

	TimerRate time.Duration = time.Second / 60  // 60hz
	ClockRate time.Duration = time.Second / 700 // 700hz

	Width  int = 64
	Height int = 32
	Area   int = Width * Height
)

const (
	Delay uint8 = 1 << iota
	Sound
	Redraw
)

var buffer [2]byte

var fontSet = []byte{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0xF0, // C
	0xE0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

type Processor struct {
	memory          [4096]byte
	v               [RegisterCount]byte
	keyState        [KeyCount]atomic.Bool
	display         [Area]byte
	stack           [16]uint16
	sp              uint8
	pc              uint16
	i               uint16
	delay           uint8
	sound           uint8
	lastTimerUpdate time.Time
}

func (p *Processor) Execute(op Opcode, info *uint8) {
	switch op.kind() {
	case 0x0:
		switch uint16(op) {
		case 0x00E0:
			clearScreen(p, info)
		case 0x00EE:
			returnFromSubroutine(p)
		default:
			panic("unknown 0x0 opcode")
		}
	case 0x1:
		jumpToLocation(p, op.nnn())
	case 0x2:
		callSubroutine(p, op.nnn())
	case 0x3:
		stepIfXEqualsNN(p, op.x(), op.nn())
	case 0x4:
		stepIfXNotEqualsNN(p, op.x(), op.nn())
	case 0x5:
		stepIfXEqualsY(p, op.x(), op.y())
	case 0x6:
		setXToNN(p, op.x(), op.nn())
	case 0x7:
		addNNToX(p, op.x(), op.nn())
	case 0x8:
		switch op.n() {
		case 0x0:
			setXToY(p, op.x(), op.y())
		case 0x1:
			orXY(p, op.x(), op.y())
		case 0x2:
			andXY(p, op.x(), op.y())
		case 0x3:
			xorXY(p, op.x(), op.y())
		case 0x4:
			addXY(p, op.x(), op.y())
		case 0x5:
			subtractYFromX(p, op.x(), op.y())
		case 0x6:
			shiftRightX(p, op.x())
		case 0x7:
			subtractXFromY(p, op.x(), op.y())
		case 0xE:
			shiftLeftX(p, op.x())
		default:
			panic("unknown 0x8 opcode")
		}
	case 0x9:
		stepIfXNotEqualsY(p, op.x(), op.y())
	case 0xA:
		setIToNNN(p, op.nnn())
	case 0xB:
		jumpWithOffset(p, op.nnn())
	case 0xC:
		setXToRandom(p, op.x(), op.nn())
	case 0xD:
		drawSprite(p, op.x(), op.y(), op.n(), info)
	case 0xE:
		switch op.nn() {
		case 0x9E:
			stepIfKeyDown(p, op.x())
		case 0xA1:
			stepIfKeyUp(p, op.x())
		default:
			panic("unknown 0xE opcode")
		}
	case 0xF:
		switch op.nn() {
		case 0x07:
			setXToDelay(p, op.x())
		case 0x0A:
			pauseUntilKeyPressed(p, op.x())
		case 0x15:
			setDelayToX(p, op.x())
		case 0x18:
			setSoundToX(p, op.x())
		case 0x1E:
			setIToX(p, op.x())
		case 0x29:
			setIToSymbol(p, op.x())
		case 0x33:
			binaryCodedDecimal(p, op.x())
		case 0x55:
			setRegistersToMemory(p, op.x())
		case 0x65:
			setMemoryToRegisters(p, op.x())
		default:
			panic("unknown 0xF opcode")
		}
	default:
		panic("unknown opcode")
	}
}

func (p *Processor) Reset() {
	*p = Processor{}

	written := p.Write(FontStartAddress, fontSet)
	if int(written) < len(fontSet) {
		panic("insufficient memory to write font set")
	}
}

func (p *Processor) Write(loc uint16, data []byte) uint16 {
	var i uint16
	for ; loc+i < 0xFFF && int(i) < len(data); i++ {
		p.memory[loc+i] = data[i]
	}
	return i
}

func (p *Processor) Read(loc uint16, data []byte) uint16 {
	var i uint16
	for ; loc+i < 0xFFF && int(i) < len(data); i++ {
		data[i] = p.memory[loc+i]
	}
	return i
}

func (p *Processor) Display() []byte {
	return p.display[:]
}

func (p *Processor) Load(b []byte) {
	written := p.Write(ProgramStartAddress, b)
	if int(written) < len(b) {
		panic("insufficient memory")
	}
	p.pc = ProgramStartAddress
}

func (p *Processor) SetKey(key uint8, value bool) {
	p.keyState[key&0x0F].Store(value)
}

func (p *Processor) DrawSprite(x, y, h uint8) {
	startX := uint16(p.v[x]) & uint16(Width-1)
	startY := uint16(p.v[y]) & uint16(Height-1)

	p.v[CarryFlag] = 0 // Reset the collision register.

	for row := range uint16(h) {
		if startY+row >= uint16(Height) {
			// Reached the bottom of the display.
			break
		}

		sprite := p.memory[p.i+row]

		for col := range uint16(8) {
			if startX+col >= uint16(Width) {
				break
			}

			if (sprite & (0x80 >> col)) != 0 {
				index := (startX + col) + ((startY + row) * uint16(Width))

				if p.display[index] == 1 {
					// Pixel was already on. This indicates a graphical object collision.
					p.v[CarryFlag] = 1 // Turn on the collision register.
				}
				p.display[index] ^= 1
			}
		}
	}
}

func (p *Processor) Register(v uint8) uint8 {
	key := v & 0xF
	return p.v[key]
}

func (p *Processor) StackDepth() int {
	return int(p.sp)
}

func (p *Processor) Index() uint16 {
	return p.i
}

func (p *Processor) ProgramCounter() uint16 {
	return p.pc
}

func (p *Processor) OpcodeAt(offset uint16) Opcode {
	read := p.Read(offset, buffer[:])
	if read < 2 {
		panic("program runaway")
	}

	// opcode is a 16bit value, comprised of two contiguous 8bit values
	// in memory, starting at the program counter
	high := uint16(buffer[0]) // high-order bits of opcode
	low := uint16(buffer[1])  // low-order bits of opcode
	return Opcode((high << 8) | low)
}

func (p *Processor) Step() uint8 {
	var info uint8

	opcode := p.OpcodeAt(p.ProgramCounter())

	p.pc += 2

	p.Execute(opcode, &info)

	if time.Since(p.lastTimerUpdate) >= TimerRate {
		if p.sound > 0 {
			p.sound--
		}

		if p.delay > 0 {
			p.delay--
		}
		p.lastTimerUpdate = time.Now()
	}

	if p.sound > 0 {
		info |= Sound
	}

	if p.delay > 0 {
		info |= Delay
	}
	return info
}
