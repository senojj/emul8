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
	"math/rand/v2"
)

type operation interface {
	Execute(*uint8)
}

type opClearScreen struct{}

func (o *opClearScreen) Execute(info *uint8) {
	for i := range cpu.display {
		cpu.display[i] = 0
	}
	*info |= Redraw
}

type opSubroutineCall struct {
	nnn uint16
}

func (o *opSubroutineCall) Execute(_ *uint8) {
	item := &node{
		value: cpu.pc,
		next:  cpu.stack,
	}
	cpu.stack = item
	cpu.pc = o.nnn
}

type opSubroutineReturn struct{}

func (o *opSubroutineReturn) Execute(_ *uint8) {
	item := cpu.stack
	if item == nil {
		panic("return from empty stack")
	}
	cpu.pc = item.value
	cpu.stack = item.next
}

type opJumpToLocation struct {
	nnn uint16
}

func (o *opJumpToLocation) Execute(_ *uint8) {
	cpu.pc = o.nnn
}

type opJumpWithOffset struct {
	nnn uint16
}

func (o *opJumpWithOffset) Execute(_ *uint8) {
	cpu.pc = o.nnn + uint16(cpu.v[0x0])
}

type opStepIfXEqualsNN struct {
	x  uint16
	nn uint16
}

func (o *opStepIfXEqualsNN) Execute(_ *uint8) {
	if cpu.v[o.x] == byte(o.nn) {
		cpu.pc += 2
	}
}

type opStepIfXNotEqualsNN struct {
	x  uint16
	nn uint16
}

func (o *opStepIfXNotEqualsNN) Execute(_ *uint8) {
	if cpu.v[o.x] != byte(o.nn) {
		cpu.pc += 2
	}
}

type opStepIfXEqualsY struct {
	x uint16
	y uint16
}

func (o *opStepIfXEqualsY) Execute(_ *uint8) {
	if cpu.v[o.x] == cpu.v[o.y] {
		cpu.pc += 2
	}
}

type opStepIfXNotEqualsY struct {
	x uint16
	y uint16
}

func (o *opStepIfXNotEqualsY) Execute(_ *uint8) {
	if cpu.v[o.x] != cpu.v[o.y] {
		cpu.pc += 2
	}
}

type opSetXToNN struct {
	x  uint16
	nn uint16
}

func (o *opSetXToNN) Execute(_ *uint8) {
	cpu.v[o.x] = byte(o.nn)
}

type opAddNNToX struct {
	x  uint16
	nn uint16
}

func (o *opAddNNToX) Execute(_ *uint8) {
	cpu.v[o.x] += byte(o.nn)
}

type opSetXToY struct {
	x uint16
	y uint16
}

func (o *opSetXToY) Execute(_ *uint8) {
	cpu.v[o.x] = cpu.v[o.y]
}

type opOrXY struct {
	x uint16
	y uint16
}

func (o *opOrXY) Execute(_ *uint8) {
	// This operation traditionally resets the carry flag.
	cpu.v[CarryFlag] = 0
	cpu.v[o.x] |= cpu.v[o.y]
}

type opAndXY struct {
	x uint16
	y uint16
}

func (o *opAndXY) Execute(_ *uint8) {
	// This operation traditionally resets the carry flag.
	cpu.v[CarryFlag] = 0
	cpu.v[o.x] &= cpu.v[o.y]
}

type opXOrXY struct {
	x uint16
	y uint16
}

func (o *opXOrXY) Execute(_ *uint8) {
	// This operation traditionally resets the carry flag.
	cpu.v[CarryFlag] = 0
	cpu.v[o.x] ^= cpu.v[o.y]
}

type opAddXY struct {
	x uint16
	y uint16
}

func (o *opAddXY) Execute(_ *uint8) {
	sum := uint16(cpu.v[o.x]) + uint16(cpu.v[o.y])
	// This operation traditionally resets the carry flag.
	cpu.v[CarryFlag] = 0
	if sum > 255 {
		cpu.v[CarryFlag] = 1
	}
	cpu.v[o.x] = byte(sum & 0xFF)
}

type opSubtractYFromX struct {
	x uint16
	y uint16
}

func (o *opSubtractYFromX) Execute(_ *uint8) {
	cpu.v[CarryFlag] = 0
	if cpu.v[o.x] >= cpu.v[o.y] {
		cpu.v[CarryFlag] = 1
	}
	cpu.v[o.x] -= cpu.v[o.y]
}

type opSubtractXFromY struct {
	x uint16
	y uint16
}

func (o *opSubtractXFromY) Execute(_ *uint8) {
	cpu.v[CarryFlag] = 0
	if cpu.v[o.y] >= cpu.v[o.x] {
		cpu.v[CarryFlag] = 1
	}
	cpu.v[o.x] = cpu.v[o.y] - cpu.v[o.x]
}

type opShiftRightX struct {
	x uint16
}

func (o *opShiftRightX) Execute(_ *uint8) {
	cpu.v[CarryFlag] = cpu.v[o.x] & 0x1
	cpu.v[o.x] >>= 1
}

type opShiftLeftX struct {
	x uint16
}

func (o *opShiftLeftX) Execute(_ *uint8) {
	cpu.v[CarryFlag] = (cpu.v[o.x] & 0x80) >> 7
	cpu.v[o.x] <<= 1
}

type opSetIToNNN struct {
	nnn uint16
}

func (o *opSetIToNNN) Execute(_ *uint8) {
	cpu.i = o.nnn
}

type opSetXToRandom struct {
	x  uint16
	nn uint16
}

func (o *opSetXToRandom) Execute(_ *uint8) {
	randomByte := byte(rand.Uint32N(256))
	cpu.v[o.x] = randomByte & byte(o.nn)
}

type opDrawSprite struct {
	x uint16
	y uint16
	n uint16
}

func (o *opDrawSprite) Execute(info *uint8) {
	DrawSprite(cpu.v[o.x], cpu.v[o.y], byte(o.n))
	*info |= Redraw
}

type opStepIfKeyDown struct {
	x uint16
}

func (o *opStepIfKeyDown) Execute(_ *uint8) {
	key := cpu.v[o.x] & 0x0F
	if cpu.keyState[key].Load() {
		cpu.pc += 2
	}
}

type opStepIfKeyUp struct {
	x uint16
}

func (o *opStepIfKeyUp) Execute(_ *uint8) {
	key := cpu.v[o.x] & 0x0F
	if !cpu.keyState[key].Load() {
		cpu.pc += 2
	}
}

type opSetXToDelay struct {
	x uint16
}

func (o *opSetXToDelay) Execute(_ *uint8) {
	cpu.v[o.x] = cpu.delay
}

type opPauseUntilKeyPressed struct {
	x uint16
}

func (o *opPauseUntilKeyPressed) Execute(_ *uint8) {
	var keyPressed bool

	for i := range uint8(len(cpu.keyState)) {
		if cpu.keyState[i].Load() {
			keyPressed = true
			cpu.v[o.x] = i
			break
		}
	}

	if !keyPressed {
		cpu.pc -= 2 // Move the program counter back, replaying the last opcode
	}
}

type opSetDelayToX struct {
	x uint16
}

func (o *opSetDelayToX) Execute(_ *uint8) {
	cpu.delay = cpu.v[o.x]
}

type opSetSoundToX struct {
	x uint16
}

func (o *opSetSoundToX) Execute(_ *uint8) {
	cpu.sound = cpu.v[o.x]
}

type opSetIToX struct {
	x uint16
}

func (o *opSetIToX) Execute(_ *uint8) {
	cpu.i += uint16(cpu.v[o.x])
}

type opSetIToSymbol struct {
	x uint16
}

func (o *opSetIToSymbol) Execute(_ *uint8) {
	digit := uint16(cpu.v[o.x] & 0x0F)
	cpu.i = FontStartAddress + (digit * 5)
}

type opBinaryCodedDecimal struct {
	x uint16
}

func (o *opBinaryCodedDecimal) Execute(_ *uint8) {
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
	val := uint32(cpu.v[o.x])

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

	cpu.memory[cpu.i] = byte((bcd >> 8) & 0xF)   // Hundreds
	cpu.memory[cpu.i+1] = byte((bcd >> 4) & 0xF) // Tens
	cpu.memory[cpu.i+2] = byte(bcd & 0xF)        // Ones
}

type opSetRegistersToMemory struct {
	x uint16
}

func (o *opSetRegistersToMemory) Execute(_ *uint8) {
	for i := uint16(0); i <= o.x; i++ {
		cpu.memory[cpu.i+i] = cpu.v[i]
	}
}

type opSetMemoryToRegisters struct {
	x uint16
}

func (o *opSetMemoryToRegisters) Execute(_ *uint8) {
	for i := uint16(0); i <= o.x; i++ {
		cpu.v[i] = cpu.memory[cpu.i+i]
	}
}

func decode(opcode uint16) operation {
	// First nibble of the opcode is the operation kind.
	kind := (opcode & 0xF000) >> 12

	// Second nibble of the opcode is the X register location.
	x := (opcode & 0x0F00) >> 8

	// Third nibble of the opcode is the Y register location.
	y := (opcode & 0x00F0) >> 4

	// Fourth nibble of the opcode is the N value.
	n := opcode & 0x000F

	// Third and fourth nibbles of the opcode combine into the NN value.
	nn := opcode & 0x00FF

	// Second, third, and fourth nibbles of the opcode combine into the NNN value.
	nnn := opcode & 0x0FFF

	switch kind {
	case 0x0:
		switch opcode {
		case 0x00E0:
			return &opClearScreen{}
		case 0x00EE:
			return &opSubroutineReturn{}
		default:
			panic("unknown 0x0 opcode")
		}
	case 0x1:
		return &opJumpToLocation{nnn}
	case 0x2:
		return &opSubroutineCall{nnn}
	case 0x3:
		return &opStepIfXEqualsNN{x, nn}
	case 0x4:
		return &opStepIfXNotEqualsNN{x, nn}
	case 0x5:
		return &opStepIfXEqualsY{x, y}
	case 0x6:
		return &opSetXToNN{x, nn}
	case 0x7:
		return &opAddNNToX{x, nn}
	case 0x8:
		switch n {
		case 0x0:
			return &opSetXToY{x, y}
		case 0x1:
			return &opOrXY{x, y}
		case 0x2:
			return &opAndXY{x, y}
		case 0x3:
			return &opXOrXY{x, y}
		case 0x4:
			return &opAddXY{x, y}
		case 0x5:
			return &opSubtractYFromX{x, y}
		case 0x6:
			return &opShiftRightX{x}
		case 0x7:
			return &opSubtractXFromY{x, y}
		case 0xE:
			return &opShiftLeftX{x}
		default:
			panic("unknown 0x8 opcode")
		}
	case 0x9:
		return &opStepIfXNotEqualsY{x, y}
	case 0xA:
		return &opSetIToNNN{nnn}
	case 0xB:
		return &opJumpWithOffset{nnn}
	case 0xC:
		return &opSetXToRandom{x, nn}
	case 0xD:
		return &opDrawSprite{x, y, n}
	case 0xE:
		switch nn {
		case 0x9E:
			return &opStepIfKeyDown{x}
		case 0xA1:
			return &opStepIfKeyUp{x}
		default:
			panic("unknown 0xE opcode")
		}
	case 0xF:
		switch nn {
		case 0x07:
			return &opSetXToDelay{x}
		case 0x0A:
			return &opPauseUntilKeyPressed{x}
		case 0x15:
			return &opSetDelayToX{x}
		case 0x18:
			return &opSetSoundToX{x}
		case 0x1E:
			return &opSetIToX{x}
		case 0x29:
			return &opSetIToSymbol{x}
		case 0x33:
			return &opBinaryCodedDecimal{x}
		case 0x55:
			return &opSetRegistersToMemory{x}
		case 0x65:
			return &opSetMemoryToRegisters{x}
		default:
			panic("unknown 0xF opcode")
		}
	default:
		panic("unknown opcode")
	}
}
