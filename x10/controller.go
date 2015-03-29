package x10

/*
#include <termios.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/select.h>
#include <sys/types.h>

void set_fd_opts(int fd) {
	fcntl(fd, F_SETFL, 0);

    struct termios portSettings;

    cfsetospeed(&portSettings, B4800);
    cfsetispeed(&portSettings, B4800);

    portSettings.c_cflag = (portSettings.c_cflag & ~CSIZE) | CS8 | B4800; // 8 databits
    portSettings.c_cflag |= (CLOCAL | CREAD);
    portSettings.c_cflag &= ~(PARENB | PARODD); // No parity
    portSettings.c_cflag &= ~CRTSCTS; // No hardware handshake
    portSettings.c_cflag &= ~CSTOPB; // 1 stopbit
    portSettings.c_iflag = IGNBRK;
    portSettings.c_iflag &= ~(IXON | IXOFF | IXANY); // No software handshake
    portSettings.c_lflag = 0;
    portSettings.c_oflag = 0;
    portSettings.c_cc[VTIME] = 1;
    portSettings.c_cc[VMIN] = 60;

    cfmakeraw(&portSettings);

    tcsetattr(fd, TCSANOW, &portSettings);
    tcflush(fd, TCIOFLUSH); // Clear IO buffer
}

*/
import "C"

import (
	"github.com/golang/glog"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
)

type Controller struct {
	tty      *os.File
	devices  []Device
	ttyMutex sync.Mutex
	finished bool
	DoneChan chan bool
}

func NewController(tty string) (*Controller, error) {
	f, err := os.OpenFile(tty, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0666)
	if err != nil {
		glog.Errorf("Controller create error: %s", err)
	} else {
		fd := C.int(f.Fd())
		if C.isatty(fd) != 1 {
			f.Close()
			glog.Errorf("Controller create error: %s", err)
		} else {
			C.set_fd_opts(fd)
		}
	}

	controller := &Controller{
		DoneChan: make(chan bool),
		tty:      f,
		finished: false,
	}

	runtime.SetFinalizer(controller, func(c *Controller) {
		c.finished = true
		c.tty.Close()
	})

	if f != nil {
		go controller.recieveData()
	}

	return controller, nil
}

func (c *Controller) AddDevice(device Device, caption string, address byte) {
	c.devices = append(c.devices, device)
	device.init(c, address, caption)
}

func (c *Controller) GetInfo() map[string]interface{} {
	res := make(map[string]interface{})

	devicesInfo := make([]interface{}, len(c.devices))
	for i := 0; i < len(c.devices); i++ {
		devInfo := c.devices[i].GetInfo()
		devInfo["address"] = AddressToString(c.devices[i].Address())
		devInfo["caption"] = c.devices[i].Caption()
		devInfo["type"] = c.devices[i].Type()
		devicesInfo[i] = devInfo
	}

	res["devices"] = devicesInfo

	return res
}

func (c *Controller) SendOn(addr string, repeats byte) bool {
	c.ttyMutex.Lock()
	defer c.ttyMutex.Unlock()

	return c.setAddr(addr, repeats) && c.sendCommand(addr, CMD_ON, repeats)
}

func (c *Controller) SendOff(addr string, repeats byte) bool {
	c.ttyMutex.Lock()
	defer c.ttyMutex.Unlock()

	return c.setAddr(addr, repeats) && c.sendCommand(addr, CMD_OFF, repeats)
}

func (c *Controller) setAddr(addr string, repeats byte) bool {
	glog.Infof("Setting address %s", addr)

	a, err := StringToAddress(addr)
	if err != nil {
		glog.Error(err)
		return false
	}

	buf := []byte{getHeader(repeats, false, false), a}

	return c.writeWithConfirm(buf)
}

func (c *Controller) sendCommand(home string, cmd byte, repeats byte) bool {
	glog.Infof("Sending command %s to %s", CommandToString(cmd), home)

	if len(home) < 1 {
		glog.Error("Empty home")
		return false
	}

	a, err := StringToAddress(string(home[0]) + "1")
	if err != nil {
		glog.Error(err)
		return false
	}

	buf := []byte{getHeader(repeats, true, false), a | (cmd & 0x0F)}

	return c.writeWithConfirm(buf)
}

func (c *Controller) recieveData() {
	fd := int(c.tty.Fd())

	for !c.finished {
		c.ttyMutex.Lock()

		var rdfs syscall.FdSet
		var timeout syscall.Timeval

		p_FD_ZERO(&rdfs)
		p_FD_SET(&rdfs, fd)

		_, err := syscall.Select(fd+1, &rdfs, nil, nil, &timeout)
		if err != nil {
			c.ttyMutex.Unlock()
			continue
		}

		buffer := make([]byte, 1)
		n, err := syscall.Read(fd, buffer)
		if err != nil || n != 1 {
			c.ttyMutex.Unlock()
			continue
		}

		c.ttyMutex.Unlock()

		//		fmt.Println(buffer)

		time.Sleep(500000000)
	}

	c.DoneChan <- true
}

func (c *Controller) writeWithConfirm(buf []byte) bool {
	glog.Infof("Start writeWithConfirm %v", buf)
	var crc byte = 0
	for _, b := range buf {
		crc += b
	}

	for try := 0; try < 3; try++ {
		n, err := c.tty.Write(buf)
		if n != len(buf) || err != nil {
			glog.Errorf("  Cannot write data on try %d", try)
			continue
		}

		checksumBuf := make([]byte, 1)
		n, err = c.tty.Read(checksumBuf)
		if n < 1 || err != nil {
			glog.Errorf("  Cannot read checksum on try %d", try)
			continue
		}

		if crc != checksumBuf[0] {
			glog.Errorf("  Invalid checksum (%d <> %d) on try %d", crc, checksumBuf[0], try)
			continue
		}

		zeroByteBuf := []byte{0}
		n, err = c.tty.Write(zeroByteBuf)
		if n < 1 || err != nil {
			glog.Errorf("  Cannot write 0 on try %d", try)
			continue
		}

		n, err = c.tty.Read(checksumBuf)
		if n < 1 || err != nil {
			glog.Errorf("  Cannot read confirm on try %d", try)
			continue
		}

		if checksumBuf[0] != 0x55 {
			glog.Errorf("  Invalid confirm (%x) on try %d", checksumBuf[0], try)
			continue
		}

		glog.Infof("Done writeWithConfirm")
		return true
	}

	return false
}

func getHeader(repeats byte, isFunction, isExtended bool) byte {
	var res byte = (repeats << 3) | (1 << 2)

	if isFunction {
		res |= 1 << 1
	}

	if isExtended {
		res |= 1
	}

	return res
}

func p_FD_SET(p *syscall.FdSet, i int) {
	p.Bits[i/64] |= 1 << uint(i) % 64
}

func p_FD_ZERO(p *syscall.FdSet) {
	for i := range p.Bits {
		p.Bits[i] = 0
	}
}
