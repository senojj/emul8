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
	"emul8/byteconv"
	"math/rand/v2"
)

func clearScreen(p *Processor, info *uint8) {
	for i := range p.display {
		p.display[i] = 0
	}
	*info |= Redraw
}

func callSubroutine(p *Processor, nnn uint16) {
	if int(p.sp) >= len(p.stack) {
		panic("stack overflow")
	}
	p.stack[p.sp] = p.pc
	p.sp++
	p.pc = nnn
}

func returnFromSubroutine(p *Processor) {
	if p.sp == 0 {
		panic("stack underflow")
	}
	p.sp--
	p.pc = p.stack[p.sp]
}

func jumpToLocation(p *Processor, nnn uint16) {
	p.pc = nnn
}

func jumpWithOffset(p *Processor, nnn uint16) {
	p.pc = nnn + uint16(p.v[0x0])
}

func stepIfXEqualsNN(p *Processor, x, nn uint8) {
	if p.v[x] == byte(nn) {
		p.pc += 2
	}
}

func stepIfXNotEqualsNN(p *Processor, x, nn uint8) {
	if p.v[x] != byte(nn) {
		p.pc += 2
	}
}

func stepIfXEqualsY(p *Processor, x, y uint8) {
	if p.v[x] == p.v[y] {
		p.pc += 2
	}
}

func stepIfXNotEqualsY(p *Processor, x, y uint8) {
	if p.v[x] != p.v[y] {
		p.pc += 2
	}
}

func setXToNN(p *Processor, x, nn uint8) {
	p.v[x] = byte(nn)
}

func addNNToX(p *Processor, x, nn uint8) {
	p.v[x] += byte(nn)
}

func setXToY(p *Processor, x, y uint8) {
	p.v[x] = p.v[y]
}

func orXY(p *Processor, x, y uint8) {
	// This operation traditionally resets the carry flag.
	p.v[CarryFlag] = 0
	p.v[x] |= p.v[y]
}

func andXY(p *Processor, x, y uint8) {
	// This operation traditionally resets the carry flag.
	p.v[CarryFlag] = 0
	p.v[x] &= p.v[y]
}

func xorXY(p *Processor, x, y uint8) {
	// This operation traditionally resets the carry flag.
	p.v[CarryFlag] = 0
	p.v[x] ^= p.v[y]
}

func addXY(p *Processor, x, y uint8) {
	sum := uint16(p.v[x]) + uint16(p.v[y])
	// This operation traditionally resets the carry flag.
	p.v[CarryFlag] = 0
	if sum > 255 {
		p.v[CarryFlag] = 1
	}
	p.v[x] = byte(sum & 0xFF)
}

func subtractYFromX(p *Processor, x, y uint8) {
	p.v[CarryFlag] = 0
	if p.v[x] >= p.v[y] {
		p.v[CarryFlag] = 1
	}
	p.v[x] -= p.v[y]
}

func subtractXFromY(p *Processor, x, y uint8) {
	p.v[CarryFlag] = 0
	if p.v[y] >= p.v[x] {
		p.v[CarryFlag] = 1
	}
	p.v[x] = p.v[y] - p.v[x]
}

func shiftRightX(p *Processor, x uint8) {
	p.v[CarryFlag] = p.v[x] & 0x1
	p.v[x] >>= 1
}

func shiftLeftX(p *Processor, x uint8) {
	p.v[CarryFlag] = (p.v[x] & 0x80) >> 7
	p.v[x] <<= 1
}

func setIToNNN(p *Processor, nnn uint16) {
	p.i = nnn
}

func setXToRandom(p *Processor, x, nn uint8) {
	randomByte := byte(rand.Uint32N(256))
	p.v[x] = randomByte & byte(nn)
}

func drawSprite(p *Processor, x, y, n uint8, info *uint8) {
	p.DrawSprite(x, y, n)
	*info |= Redraw
}

func stepIfKeyDown(p *Processor, x uint8) {
	key := p.v[x] & 0x0F
	if p.keyState[key].Load() {
		p.pc += 2
	}
}

func stepIfKeyUp(p *Processor, x uint8) {
	key := p.v[x] & 0x0F
	if !p.keyState[key].Load() {
		p.pc += 2
	}
}

func setXToDelay(p *Processor, x uint8) {
	p.v[x] = p.delay
}

func pauseUntilKeyPressed(p *Processor, x uint8) {
	var keyPressed bool

	for i := range uint8(len(p.keyState)) {
		if p.keyState[i].Load() {
			keyPressed = true
			p.v[x] = i
			break
		}
	}

	if !keyPressed {
		p.pc -= 2 // Move the program counter back, replaying the last opcode
	}
}

func setDelayToX(p *Processor, x uint8) {
	p.delay = p.v[x]
}

func setSoundToX(p *Processor, x uint8) {
	p.sound = p.v[x]
}

func setIToX(p *Processor, x uint8) {
	p.i += uint16(p.v[x])
}

func setIToSymbol(p *Processor, x uint8) {
	digit := uint16(p.v[x] & 0x0F)
	p.i = FontStartAddress + (digit * 5)
}

func binaryCodedDecimal(p *Processor, x uint8) {
	// Takes the number in register VX (which is one byte, so it can be any number from
	// 0 to 255) and converts it to three decimal digits, storing these digits in memory
	// at the address in the index register I. For example, if VX contains 156 (or 9C in
	// hexadecimal), it would put the number 1 at the address in I, 5 in address I + 1,
	// and 6 in address I + 2.

	// Double Dabble algorithm.
	//
	// Converts binary numbers to Binary-Coded Decimal (BCD) by repeatedly shifting
	// and adding 3 to nibbles that exceed 4, effectively performing a base conversion
	// in hardware. It starts with a binary input and an empty BCD register, iterating
	// for each bit, left-shifting the combined register, and injecting the next binary
	// bit, adding 3 to any BCD nibble >= 5 to handle carries, making it efficient for
	// digital displays.
	//
	// "Double": Each left shift effectively multiplies the BCD digits by 2.
	//
	// "Dabble": Adding 3 when a nibble hits 5 or more ensures that when it's shifted,
	//           it carries over correctly (e.g., 5 becomes 8, shift makes it 16, which
	//           is 10 in decimal, correctly carrying to the next place).
	//
	// The idea is that this implementation should be more efficient than integer division
	// and modulo operations.
	var bcd uint32

	// Fetch the value from register VX as a 32bit integer.
	val := uint32(p.v[x])

	// Iterate 8 times (once for each bit of the input byte)
	// Check each BCD nibble. If >= 5, add 3.
	for i := range 8 {
		// Ones (bits 0-3)
		if (bcd & 0x00F) >= 5 {
			bcd += 3
		}

		// Tens (bits 4-7)
		if (bcd & 0x0F0) >= 0x050 {
			bcd += 0x030
		}

		// Hundreds (bits 8-11)
		if (bcd & 0xF00) >= 0x500 {
			bcd += 0x300
		}

		// Shift BCD left by 1, and pull in the next bit from `val`
		bcd = (bcd << 1) | ((val >> (7 - i)) & 1)
	}

	p.memory[p.i] = byte((bcd >> 8) & 0xF)   // Hundreds
	p.memory[p.i+1] = byte((bcd >> 4) & 0xF) // Tens
	p.memory[p.i+2] = byte(bcd & 0xF)        // Ones
}

func setRegistersToMemory(p *Processor, x uint8) {
	for i := uint8(0); i <= x; i++ {
		p.memory[p.i+uint16(i)] = p.v[i]
	}
}

func setMemoryToRegisters(p *Processor, x uint8) {
	for i := uint8(0); i <= x; i++ {
		p.v[i] = p.memory[p.i+uint16(i)]
	}
}

type Opcode uint16

func (o Opcode) kind() uint8 {
	return uint8((uint16(o) & 0xF000) >> 12)
}

func (o Opcode) x() uint8 {
	return uint8((uint16(o) & 0x0F00) >> 8)
}

func (o Opcode) y() uint8 {
	return uint8((uint16(o) & 0x00F0) >> 4)
}

func (o Opcode) n() uint8 {
	return uint8(uint16(o) & 0x000F)
}

func (o Opcode) nn() uint8 {
	return uint8(uint16(o) & 0x00FF)
}

func (o Opcode) nnn() uint16 {
	return uint16(o) & 0x0FFF
}

func u16toh(i uint16, n int) string {
	return byteconv.Btoh(byteconv.U16tob(i), n)
}

func u8toh(i uint8, n int) string {
	return byteconv.Btoh(byteconv.U16tob(uint16(i)), n)
}

func (op Opcode) String() string {
	var str string

	switch op.kind() {
	case 0x0:
		switch uint16(op) {
		case 0x00E0:
			str = "CLS"
		case 0x00EE:
			str = "RET"
		default:
			panic("unknown 0x0 opcode")
		}
	case 0x1:
		str = "JP " + u16toh(op.nnn(), 3)
	case 0x2:
		str = "CALL " + u16toh(op.nnn(), 3)
	case 0x3:
		str = "SE V" + u8toh(op.x(), 1) + ", " + u8toh(op.nn(), 2)
	case 0x4:
		str = "SNE V" + u8toh(op.x(), 1) + ", " + u8toh(op.nn(), 2)
	case 0x5:
		str = "SE V" + u8toh(op.x(), 1) + ", V" + u8toh(op.y(), 1)
	case 0x6:
		str = "LD V" + u8toh(op.x(), 1) + ", " + u8toh(op.nn(), 2)
	case 0x7:
		str = "ADD V" + u8toh(op.x(), 1) + ", " + u8toh(op.nn(), 2)
	case 0x8:
		switch op.n() {
		case 0x0:
			str = "LD V" + u8toh(op.x(), 1) + ", V" + u8toh(op.y(), 1)
		case 0x1:
			str = "OR V" + u8toh(op.x(), 1) + ", V" + u8toh(op.y(), 1)
		case 0x2:
			str = "AND V" + u8toh(op.x(), 1) + ", V" + u8toh(op.y(), 1)
		case 0x3:
			str = "XOR V" + u8toh(op.x(), 1) + ", V" + u8toh(op.y(), 1)
		case 0x4:
			str = "ADD V" + u8toh(op.x(), 1) + ", V" + u8toh(op.y(), 1)
		case 0x5:
			str = "SUB V" + u8toh(op.x(), 1) + ", V" + u8toh(op.y(), 1)
		case 0x6:
			str = "SHR V" + u8toh(op.x(), 1)
		case 0x7:
			str = "SUBN V" + u8toh(op.x(), 1) + ", V" + u8toh(op.y(), 1)
		case 0xE:
			str = "SHL V" + u8toh(op.x(), 1)
		default:
			panic("unknown 0x8 opcode")
		}
	case 0x9:
		str = "SNE V" + u8toh(op.x(), 1) + ", V" + u8toh(op.y(), 1)
	case 0xA:
		str = "LD I, " + u16toh(op.nnn(), 3)
	case 0xB:
		str = "JP V0, " + u16toh(op.nnn(), 3)
	case 0xC:
		str = "RND V" + u8toh(op.x(), 1) + ", " + u8toh(op.nn(), 2)
	case 0xD:
		str = "DRW V" + u8toh(op.x(), 1) + ", V" + u8toh(op.y(), 1) + ", " + u8toh(op.nn(), 2)
	case 0xE:
		switch op.nn() {
		case 0x9E:
			str = "SKP V" + u8toh(op.x(), 1)
		case 0xA1:
			str = "SKNP V" + u8toh(op.x(), 1)
		default:
			panic("unknown 0xE opcode")
		}
	case 0xF:
		switch op.nn() {
		case 0x07:
			str = "LD V" + u8toh(op.x(), 1) + ", DT"
		case 0x0A:
			str = "LD V" + u8toh(op.x(), 1) + ", K"
		case 0x15:
			str = "LD DT, V" + u8toh(op.x(), 1)
		case 0x18:
			str = "LD ST, V" + u8toh(op.x(), 1)
		case 0x1E:
			str = "ADD I, V" + u8toh(op.x(), 1)
		case 0x29:
			str = "LD F, V" + u8toh(op.x(), 1)
		case 0x33:
			str = "LD B, V" + u8toh(op.x(), 1)
		case 0x55:
			str = "LD [I], V" + u8toh(op.x(), 1)
		case 0x65:
			str = "LD V" + u8toh(op.x(), 1) + ", [I]"
		default:
			panic("unknown 0xF opcode")
		}
	default:
		panic("unknown opcode")
	}
	return str
}
