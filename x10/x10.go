package x10

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	CMD_ALL_UNITS_OFF byte = iota
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

var addrBits = [...]byte{12, 4, 2, 10, 14, 6, 0, 8, 13, 5, 3, 11, 15, 7, 1, 9}
var bitsToPos map[byte]byte
var cmdsStrings = [...]string{"ALL_UNITS_OFF", "ALL_LIGHTS_ON", "ON", "OFF", "DIM", "BRIGHT", "ALL_LIGHTS_OFF",
	"EXTENDED", "HAIL_REQUEST", "HAIL_ACKNOWLEDGE", "EXT3", "EXT4", "EXT2", "STATUS_ON", "STATUS_OFF", "STATUS_REQUEST"}

func StringToAddress(s string) (byte, error) {
	s = strings.ToUpper(s)
	if len(s) >= 2 {
		devId, err := strconv.Atoi(s[1:])
		if err == nil && s[0] >= 'A' && s[0] <= 'P' && devId >= 1 && devId <= 16 {
			return addrBits[s[0]-'A']<<4 + addrBits[(devId)-1], nil
		}
	}
	return 0, fmt.Errorf("Invalid address %s", s)
}

func AddressToString(addr byte) string {
	if bitsToPos == nil {
		bitsToPos = make(map[byte]byte)
		for i, v := range addrBits {
			bitsToPos[v] = byte(i)
		}
	}

	return string('A'+bitsToPos[addr>>4]) + strconv.Itoa(int(bitsToPos[addr&0x0F])+1)
}

func CommandToString(cmd byte) string {
	return cmdsStrings[cmd&0x0F]
}
