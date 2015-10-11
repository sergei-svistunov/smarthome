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
	tty               *os.File
	sendRepeats       byte
	devices           map[X10Addr]IDevice
	ttyMutex          sync.Mutex
	finished          bool
	addrBuf           []X10Addr
	cmdAddrBuf        []X10Addr
	onUpdateCallbacks []func(device IDevice)
	DoneChan          chan bool
}

func NewController(tty string, sendRepeats byte) (*Controller, error) {
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
		DoneChan:    make(chan bool),
		tty:         f,
		devices:     make(map[X10Addr]IDevice),
		sendRepeats: sendRepeats,
		finished:    false,
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

func (c *Controller) AddDevice(device IDevice, caption string, address X10Addr) {
	c.devices[address] = device
	device.init(c, address, caption)
}

func (c *Controller) RegisterOnUpdateCallback(callback func(device IDevice)) {
	c.onUpdateCallbacks = append(c.onUpdateCallbacks, callback)
}

func (c *Controller) GetInfo() map[string]interface{} {
	res := make(map[string]interface{})

	devicesInfo := make([]interface{}, 0, len(c.devices))
	for _, device := range c.devices {
		devicesInfo = append(devicesInfo, c.GetDeviceInfo(device))
	}

	res["devices"] = devicesInfo

	return res
}

func (c *Controller) GetDeviceInfo(device IDevice) map[string]interface{} {
	devInfo := device.GetInfo()
	devInfo["address"] = device.Address().String()
	devInfo["caption"] = device.Caption()
	devInfo["type"] = device.Type()

	return devInfo
}

func (c *Controller) SendStatusRequest(addr string) bool {
	c.ttyMutex.Lock()
	defer c.ttyMutex.Unlock()

	return c.setAddr(addr) && c.sendCommand(addr, CMD_STATUS_REQUEST)
}

func (c *Controller) SendOn(addr string) bool {
	c.ttyMutex.Lock()
	defer c.ttyMutex.Unlock()

	return c.setAddr(addr) && c.sendCommand(addr, CMD_ON)
}

func (c *Controller) SendOff(addr string) bool {
	c.ttyMutex.Lock()
	defer c.ttyMutex.Unlock()

	return c.setAddr(addr) && c.sendCommand(addr, CMD_OFF)
}

func (c *Controller) SendPresetDim(addr string, volume uint8) bool {
	c.ttyMutex.Lock()
	defer c.ttyMutex.Unlock()

	if volume > 0x3f {
		volume = 0x3f
	}

	return c.sendExtendedCommand(addr, ECMD_PRESET_DIM, volume)
}

func (c *Controller) setAddr(addr string) bool {
	glog.Infof("Setting address %s", addr)

	a, err := StringToAddress(addr)
	if err != nil {
		glog.Error(err)
		return false
	}

	buf := []byte{
		getHeader(c.sendRepeats, false, false),
		byte(a),
	}

	if c.writeWithConfirm(buf) {
		c.cmdAddrBuf = append(c.cmdAddrBuf, a)
		return true
	}

	return false
}

func (c *Controller) sendCommand(home string, cmd X10Command) bool {
	glog.Infof("Sending command %s to %s", cmd, home)

	if len(home) < 1 {
		glog.Error("Empty home")
		return false
	}

	a, err := StringToAddress(string(home[0]) + "1")
	if err != nil {
		glog.Error(err)
		return false
	}

	buf := []byte{
		getHeader(c.sendRepeats, true, false),
		byte(a&0xf0) | (byte(cmd & 0x0f)),
	}

	if c.writeWithConfirm(buf) {
		c.notify(c.cmdAddrBuf, cmd, 0)
		c.cmdAddrBuf = c.cmdAddrBuf[:0]
		return true
	}

	return false
}

func (c *Controller) sendExtendedCommand(address string, cmd X10ExtendedCommand, data uint8) bool {
	glog.Infof("Sending extended command %s to %s", cmd, address)

	a, err := StringToAddress(address)
	if err != nil {
		glog.Error(err)
		return false
	}

	buf := []byte{
		getHeader(c.sendRepeats, true, true),
		byte(a&0xf0) | (byte(CMD_EXTENDED & 0x0f)),
		byte(a & 0x0f),
		data,
		byte(cmd),
	}

	if c.writeWithConfirm(buf) {
		c.notifyExt(a, cmd, data)
		return true
	}

	return false
}

func (c *Controller) notify(addresses []X10Addr, cmd X10Command, data uint8) {
	for _, addr := range addresses {
		device, exists := c.devices[addr]
		if exists {
			device.notify(cmd, data)
			for _, cb := range c.onUpdateCallbacks {
				cb(device)
			}
		}
	}
}

func (c *Controller) notifyExt(addr X10Addr, cmd X10ExtendedCommand, data uint8) {
	device, exists := c.devices[addr]
	if exists {
		device.notifyExt(cmd, data)
		for _, cb := range c.onUpdateCallbacks {
			cb(device)
		}
	}
}

func (c *Controller) recieveData() {
	fd := int(c.tty.Fd())

	for !c.finished {
		time.Sleep(500000000)

		c.ttyMutex.Lock()

		var rdfs syscall.FdSet
		timeout := syscall.Timeval{0, 0}

		p_FD_ZERO(&rdfs)
		p_FD_SET(&rdfs, fd)

		n, err := syscall.Select(fd+1, &rdfs, nil, nil, &timeout)
		if n < 1 || err != nil {
			c.ttyMutex.Unlock()
			continue
		}

		buffer := make([]byte, 1)
		n, err = syscall.Read(fd, buffer)
		if err != nil || n != 1 || buffer[0] != 0x5a {
			c.ttyMutex.Unlock()
			continue
		}

		glog.Info("X10 controller has data")

		_, err = syscall.Write(fd, []byte{0xc3})
		if err != nil {
			glog.Errorf("Cannot write confirm to controller: %s", err)
			c.ttyMutex.Unlock()
			continue
		}

		n, err = syscall.Read(fd, buffer)
		if err != nil || n != 1 {
			glog.Errorf("Cannot read message length: %s", err)
			c.ttyMutex.Unlock()
			continue
		}

		messageBuffer := make([]byte, 0, buffer[0])
		needRead := int(buffer[0])
		for needRead > 0 {
			tmpBuf := make([]byte, needRead)

			n, err = syscall.Read(fd, tmpBuf)
			if err != nil {
				glog.Errorf("Cannot read message: %s", err)
				c.ttyMutex.Unlock()
				continue
			}
			messageBuffer = append(messageBuffer, tmpBuf[:n]...)
			needRead -= n
		}

		c.ttyMutex.Unlock()

		glog.Infof("messageBuffer: %v", messageBuffer)
		for i := 0; i < len(messageBuffer)-1; i++ {
			if messageBuffer[0]&(1<<uint(i)) > 0 { //command
				command := X10Command(messageBuffer[i+1] & 0x0f)
				switch command {
				case CMD_DIM, CMD_BRIGHT:
					glog.Infof("Recived command %s with value %d", command, messageBuffer[i+2])
					c.notify(c.addrBuf, command, messageBuffer[i+2])
					i++
				default:
					glog.Infof("Recived command %s", command)
					c.notify(c.addrBuf, command, 0)
				}

				c.addrBuf = c.addrBuf[:0]
			} else { //address
				c.addrBuf = append(c.addrBuf, X10Addr(messageBuffer[i+1]))
				glog.Info("Add address", c.addrBuf)
			}
		}
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

func getHeader(repeats uint8, isFunction, isExtended bool) uint8 {
	var res uint8 = (repeats << 3) | (1 << 2)

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
