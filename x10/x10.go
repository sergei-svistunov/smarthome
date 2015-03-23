package x10

import (
	"errors"
	"strconv"
	"strings"
)

var addrBits = [...]byte{12, 4, 2, 10, 14, 6, 0, 8, 13, 5, 3, 11, 15, 7, 1, 9}

func StringToAddress(s string) (byte, error) {
	s = strings.ToUpper(s)
	if len(s) >= 2 {
		devId, err := strconv.Atoi(s[1:])
		if err == nil && s[0] >= 'A' && s[0] <= 'P' && devId >= 1 && devId <= 16 {
			return addrBits[s[0]-'A']<<4 + addrBits[(devId)-1], nil
		}
	}
	return 0, errors.New("Invalid address")
}
