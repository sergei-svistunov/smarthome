package x10

import (
	"fmt"
	"strconv"
	"strings"
)

type X10Addr uint8
type X10Command uint8
type X10ExtendedCommand uint8

type IDevice interface {
	init(controller *Controller, addr X10Addr, caption string)
	notify(command X10Command, data uint8)
	notifyExt(command X10ExtendedCommand, data uint8)
	Address() X10Addr
	Caption() string
	Type() string
	GetInfo() map[string]interface{}
}

const (
	CMD_ALL_UNITS_OFF X10Command = iota
	CMD_ALL_LIGHTS_ON
	CMD_ON
	CMD_OFF
	CMD_DIM
	CMD_BRIGHT
	CMD_ALL_LIGHTS_OFF
	CMD_EXTENDED
	CMD_HAIL_REQUEST
	CMD_HAIL_ACKNOWLEDGE
	CMD_EXT3
	CMD_EXT4
	CMD_EXT2
	CMD_STATUS_ON
	CMD_STATUS_OFF
	CMD_STATUS_REQUEST
)

const (
	ECMD_PRESET_DIM X10ExtendedCommand = 0x31
)

var addrBits = [...]byte{12, 4, 2, 10, 14, 6, 0, 8, 13, 5, 3, 11, 15, 7, 1, 9}
var bitsToPos, posToBits map[byte]byte
var cmdsStrings = [...]string{"ALL_UNITS_OFF", "ALL_LIGHTS_ON", "ON", "OFF", "DIM", "BRIGHT", "ALL_LIGHTS_OFF",
	"EXTENDED", "HAIL_REQUEST", "HAIL_ACKNOWLEDGE", "EXT3", "EXT4", "EXT2", "STATUS_ON", "STATUS_OFF", "STATUS_REQUEST"}

func init() {
	bitsToPos = make(map[byte]byte)
	posToBits = make(map[byte]byte)
	for i, v := range addrBits {
		bitsToPos[v] = byte(i)
		posToBits[byte(i)] = v
	}
}

func (a X10Addr) String() string {
	return string('A'+posToBits[byte(a)>>4]) + strconv.Itoa(int(posToBits[byte(a)&0x0f])+1)
}

func (c X10Command) String() string {
	return cmdsStrings[byte(c)&0x0f]
}

func (c X10ExtendedCommand) String() string {
	switch c {
	case ECMD_PRESET_DIM:
		return "PRESET_DIM"
	default:
		return "UNKNOWN"
	}
}

func StringToAddress(s string) (X10Addr, error) {
	s = strings.ToUpper(s)
	if len(s) >= 2 {
		devId, err := strconv.Atoi(s[1:])
		if err == nil && s[0] >= 'A' && s[0] <= 'P' && devId >= 1 && devId <= 16 {
			return X10Addr(bitsToPos[byte(s[0]-'A')]<<4 + bitsToPos[byte(devId-1)]), nil
		}
	}
	return 0, fmt.Errorf("Invalid address %s", s)
}
