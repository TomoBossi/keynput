package main

import (
	"time"

	"github.com/tomobossi/keynput"
)

func main() {
	keyboard, err := keynput.NewKeyboard("keypresses_demo")
	if err != nil {
		panic(err)
	}
	defer keyboard.Close()

	for _, keycode := range []uint16{
		keynput.KEY_E,
		keynput.KEY_C,
		keynput.KEY_H,
		keynput.KEY_O,
		keynput.KEY_SPACE,
		keynput.KEY_H,
		keynput.KEY_E,
		keynput.KEY_L,
		keynput.KEY_L,
		keynput.KEY_O,
		keynput.KEY_ENTER,
	} {
		time.Sleep(time.Millisecond * 100)
		err := keyboard.KeyPress(keycode)
		if err != nil {
			panic(err)
		}
	}
}
