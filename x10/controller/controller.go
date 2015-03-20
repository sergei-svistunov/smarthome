package controller

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
	"errors"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"

	"fmt"
)

type Controller struct {
	tty      *os.File
	ttyMutex sync.Mutex
	finished bool
	DoneChan chan bool
}

func NewController(tty string) (*Controller, error) {
	f, err := os.OpenFile(tty, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0666)
	if err != nil {
		return nil, err
		fmt.Println("Controller create error")
	}

	fd := C.int(f.Fd())
	if C.isatty(fd) != 1 {
		f.Close()
		return nil, errors.New("File is not a tty")
	}

	C.set_fd_opts(fd)

	controller := new(Controller)
	runtime.SetFinalizer(controller, func(c *Controller) {
		fmt.Println("Controller destructor")
		c.finished = true
		c.tty.Close()
	})

	controller.DoneChan = make(chan bool)
	controller.tty = f
	controller.finished = false

	go controller.recieveData()

	return controller, nil
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
		
		fmt.Println(buffer)

		time.Sleep(500000000)
	}
	
	c.DoneChan <- true
}

func (c *Controller) Test() {
	fmt.Println("Controller test")
}

func p_FD_SET(p *syscall.FdSet, i int) {
	p.Bits[i/64] |= 1 << uint(i) % 64
}

func p_FD_ZERO(p *syscall.FdSet) {
	for i := range p.Bits {
		p.Bits[i] = 0
	}
}
