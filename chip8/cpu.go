package chip8

import (
	"sync/atomic"
	"time"
)

const (
	RegisterCount       = 16
	KeyCount            = 16
	FontStartAddress    = 0x50
	LastAddress         = 0xFFE
	ProgramStartAddress = 0x200
	CarryFlag           = 0xF

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

type node struct {
	value uint16
	next  *node
}

type processor struct {
	memory          [4096]byte
	v               [RegisterCount]byte
	keyState        [KeyCount]atomic.Bool
	display         [Area]byte
	stack           *node
	pc              uint16
	i               uint16
	delay           uint8
	sound           uint8
	lastTimerUpdate time.Time
}

var cpu processor

func init() {
	Reset()
}

func Reset() {
	for i := range len(cpu.memory) {
		cpu.memory[i] = 0
	}

	for i := range len(cpu.v) {
		cpu.v[i] = 0
	}

	for i := range len(cpu.keyState) {
		cpu.keyState[i].Store(false)
	}

	for i := range len(cpu.display) {
		cpu.display[i] = 0
	}

	cpu.stack = nil
	cpu.pc = ProgramStartAddress
	cpu.i = 0
	cpu.delay = 0
	cpu.sound = 0

	var t time.Time
	cpu.lastTimerUpdate = t

	written := Write(FontStartAddress, fontSet)
	if int(written) < len(fontSet) {
		panic("insufficient memory to write font set")
	}
}

func Write(loc uint16, data []byte) uint16 {
	var i uint16
	for ; loc+i < 0xFFF && int(i) < len(data); i++ {
		cpu.memory[loc+i] = data[i]
	}
	return i
}

func Read(loc uint16, data []byte) uint16 {
	var i uint16
	for ; loc+i < 0xFFF && int(i) < len(data); i++ {
		data[i] = cpu.memory[loc+i]
	}
	return i
}

func Display() []byte {
	return cpu.display[:]
}

func Load(b []byte) {
	written := Write(ProgramStartAddress, b)
	if int(written) < len(b) {
		panic("insufficient memory")
	}
}

func SetKey(key uint8, value bool) {
	cpu.keyState[key&0x0F].Store(value)
}

func DrawSprite(x, y, h byte) {
	startX := uint16(x) & uint16(Width-1)
	startY := uint16(y) & uint16(Height-1)

	cpu.v[CarryFlag] = 0 // Reset the collision register.

	for row := range uint16(h) {
		if startY+row >= uint16(Height) {
			// Reached the bottom of the display.
			break
		}

		sprite := cpu.memory[cpu.i+row]

		for col := range uint16(8) {
			if startX+col >= uint16(Width) {
				break
			}

			if (sprite & (0x80 >> col)) != 0 {
				index := (startX + col) + ((startY + row) * uint16(Width))

				if cpu.display[index] == 1 {
					// Pixel was already on. This indicates a graphical object collision.
					cpu.v[CarryFlag] = 1 // Turn on the collision register.
				}
				cpu.display[index] ^= 1
			}
		}
	}
}

func ProgramCounter() uint16 {
	return cpu.pc
}

func Opcode(offset uint16) uint16 {
	var buffer [2]byte
	read := Read(offset, buffer[:])
	if read < 2 {
		panic("program runaway")
	}

	// opcode is a 16bit value, comprised of two contiguous 8bit values
	// in memory, starting at the program counter
	high := uint16(buffer[0]) // high-order bits of opcode
	low := uint16(buffer[1])  // low-order bits of opcode
	return (high << 8) | low
}

func Step() uint8 {
	var info uint8

	opcode := Opcode(cpu.pc)

	cpu.pc += 2

	decode(opcode).Execute(&info)

	if time.Since(cpu.lastTimerUpdate) >= TimerRate {
		if cpu.sound > 0 {
			cpu.sound--
		}

		if cpu.delay > 0 {
			cpu.delay--
		}
		cpu.lastTimerUpdate = time.Now()
	}

	if cpu.sound > 0 {
		info |= Sound
	}

	if cpu.delay > 0 {
		info |= Delay
	}
	return info
}
