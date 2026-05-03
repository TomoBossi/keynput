package keynput

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
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

func Open(name string) (*Keyboard, error) {
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
			busType: BUS_USB,
			vendor:  0xB0CA,
			product: 0xDEAD,
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

	time.Sleep(time.Millisecond * 300)

	return &Keyboard{
		devNode: devNode,
		device:  device,
	}, nil
}

func (k *Keyboard) Close() error {
	if k.devNode == nil {
		return nil
	}
	err := close(k.devNode)
	k.devNode = nil
	return err
}

func (k *Keyboard) KeyPress(keycode uint16) error {
	if keycode < 1 || keycode > keyMax {
		return fmt.Errorf("code %d out of range [1, %d]", keycode, keyMax)
	}

	return sendKeyPressEvent(k.devNode, keycode)
}

func sendKeyPressEvent(devNode *os.File, keycode uint16) error {
	err := sendKeyEvent(devNode, keycode, BTN_PRESSED)
	if err != nil {
		return err
	}

	return sendKeyEvent(devNode, keycode, BTN_RELEASED)
}

func sendKeyEvent(devNode *os.File, keycode uint16, value uint32) error {
	err := sendEvent(
		inputEvent{
			time:  timeval{sec: 0, usec: 0},
			typ:   EV_KEY,
			code:  keycode,
			value: value,
		},
		devNode,
	)
	if err != nil {
		return err
	}

	return sendSyncEvent(devNode)
}

func sendSyncEvent(devNode *os.File) error {
	return sendEvent(
		inputEvent{
			time:  timeval{sec: 0, usec: 0},
			typ:   EV_SYN,
			code:  SYN_REPORT,
			value: 0,
		},
		devNode,
	)

}

func sendEvent(event inputEvent, devNode *os.File) error {
	bytes, err := inputEventBytes(event)
	if err != nil {
		return err
	}

	_, err = devNode.Write(bytes)
	return err
}

func inputEventBytes(event inputEvent) ([]byte, error) {
	buffer := bytes.NewBuffer(make([]byte, 0, inputEventSize))
	err := binary.Write(buffer, binary.LittleEndian, event)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
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
