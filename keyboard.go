package keynput

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	inputEventSize = 24
	keyMax         = 248
	uiMaxNameSize  = 80
	uiDevDestroy   = 0x5502
	uiDevCreate    = 0x5501
	uiSetEvBit     = 0x40045564
	uiSetKeyBit    = 0x40045565
)

type inputEvent struct {
	time  timeval
	typ   uint16
	code  uint16
	value uint32
} // 24 bytes

type timeval struct {
	sec  uint64
	usec uint64
}

type inputID struct {
	busType uint16
	vendor  uint16
	product uint16
	version uint16
}

type uinputDevice struct {
	name [uiMaxNameSize]byte
	id   inputID
	_    int32
	_    [64]int32
	_    [64]int32
	_    [64]int32
	_    [64]int32
}

type Keyboard struct {
	devNode *os.File
	device  uinputDevice
}

func NewKeyboard(name string) (*Keyboard, error) {
	if len(name) == 0 || len(name) > uiMaxNameSize {
		return nil, fmt.Errorf("name must be more than 0 and less than %d characters", uiMaxNameSize)
	}
	var fixedSizeName [uiMaxNameSize]byte
	copy(fixedSizeName[:], []byte(name))

	devNode, err := os.OpenFile("/dev/uinput", syscall.O_WRONLY, 0)
	if err != nil {
		return nil, err
	}

	err = ioctl(devNode, uiSetEvBit, uintptr(EV_KEY))
	if err != nil {
		return nil, err
	}

	for i := 0; i <= keyMax; i++ {
		err = ioctl(devNode, uiSetKeyBit, uintptr(i))
		if err != nil {
			return nil, err
		}
	}

	device := uinputDevice{
		name: fixedSizeName,
		id: inputID{
			busType: 0x03,
			vendor:  0x4711,
			product: 0x0815,
			version: 1,
		},
	}

	buffer := new(bytes.Buffer)
	err = binary.Write(buffer, binary.LittleEndian, device)
	if err != nil {
		return nil, err
	}

	_, err = devNode.Write(buffer.Bytes())
	if err != nil {
		return nil, err
	}

	err = ioctl(devNode, uiDevCreate, uintptr(0))
	if err != nil {
		return nil, err
	}

	err = awaitDevice(device)
	if err != nil {
		return nil, err
	}

	return &Keyboard{
		devNode: devNode,
		device:  device,
	}, nil
}

func awaitDevice(device uinputDevice) error {
	devNodePath := ""
	for len(devNodePath) == 0 {
		devices, err := os.ReadFile("/proc/bus/input/devices")
		if err != nil {
			return err
		}

		re := regexp.MustCompile(`(?s)Bus=(.*) Vendor=(.*) Product=(.*) Version=(.*)\nN: Name="(.*)"\nP: Phys=\n.*H: Handlers=.*event(\d+)`)
		for dev := range strings.SplitSeq(string(devices), "\nI: ") {

			matches := re.FindStringSubmatch(dev)
			if matches == nil {
				continue
			}

			bus, err := strconv.ParseInt(matches[1], 16, 64)
			if err != nil {
				return err
			} else if uint16(bus) != device.id.busType {
				continue
			}

			vendor, err := strconv.ParseInt(matches[2], 16, 64)
			if err != nil {
				return err
			} else if uint16(vendor) != device.id.vendor {
				continue
			}

			product, err := strconv.ParseInt(matches[3], 16, 64)
			if err != nil {
				return err
			} else if uint16(product) != device.id.product {
				continue
			}

			version, err := strconv.ParseInt(matches[4], 16, 64)
			if err != nil {
				return err
			} else if uint16(version) != device.id.version {
				continue
			}

			var fixedSizeName [uiMaxNameSize]byte
			copy(fixedSizeName[:], []byte(matches[5]))
			if fixedSizeName != device.name {
				continue
			}

			devNodePath = fmt.Sprintf("/dev/input/event%s", matches[6])
			break
		}
		time.Sleep(time.Millisecond)
	}

	for {
		if _, err := os.Stat(devNodePath); err == nil {
			break
		}
		time.Sleep(time.Millisecond)
	}

	return nil
}

func (k *Keyboard) KeyPress(keycode uint16) error {
	if keycode < 1 || keycode > keyMax {
		return fmt.Errorf("code %d out of range [1, %d]", keycode, keyMax)
	}

	err := sendKeyEvent(k.devNode, keycode, BTN_PRESSED)
	if err != nil {
		return err
	}

	err = sendKeyEvent(k.devNode, keycode, BTN_RELEASED)
	if err != nil {
		return err
	}

	return nil
}

func sendKeyEvent(devNode *os.File, key uint16, value uint32) error {
	buffer, err := inputEventToBuffer(
		inputEvent{
			time:  timeval{sec: 0, usec: 0},
			typ:   EV_KEY,
			code:  key,
			value: value,
		},
	)
	if err != nil {
		return err
	}

	_, err = devNode.Write(buffer)
	if err != nil {
		return err
	}

	return syncEvent(devNode)
}

func syncEvent(devNode *os.File) error {
	buffer, err := inputEventToBuffer(inputEvent{
		time:  timeval{sec: 0, usec: 0},
		typ:   EV_SYN,
		code:  SYN_REPORT,
		value: 0})
	if err != nil {
		return err
	}
	_, err = devNode.Write(buffer)
	return err
}

func inputEventToBuffer(event inputEvent) ([]byte, error) {
	buffer := bytes.NewBuffer(make([]byte, 0, inputEventSize))
	err := binary.Write(buffer, binary.LittleEndian, event)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (k *Keyboard) Close() error {
	return close(k.devNode)
}

func close(devNode *os.File) error {
	err := ioctl(devNode, uiDevDestroy, uintptr(0))
	if err != nil {
		return err
	}
	return devNode.Close()
}

func ioctl(devNode *os.File, cmd, ptr uintptr) error {
	_, _, errNo := syscall.Syscall(syscall.SYS_IOCTL, devNode.Fd(), cmd, ptr)
	if errNo != 0 {
		err := close(devNode)
		if err != nil {
			return err
		} else {
			return errNo
		}
	}
	return nil
}
