# keynput
`uinput` but tiny and for **key**board i**nput**s only.
## Why?
Because a while back I used [`uinput`](https://github.com/bendahl/uinput) to make [`typeworm`](https://github.com/TomoBossi/typeworm), and since then I've been wanting to basically make `uinput` myself.
## What?
This is a minimal `uinput`-like module, but it only works for keyboard devices. You are probably better off using some other more robust implementation, even if deprecated. This was made just for fun.
## How?
Be on Linux. Install the module, import and use it in your program, then compile and run. Make sure you be on Linux first.

```go
func main() {
	keyboard, err := keynput.NewKeyboard("virtual_keyboard")
	if err != nil {
		panic(err)
	}
	defer keyboard.Close()
	
	// send keypress events
	err := keyboard.KeyPress(keynput.KEY_E)
	if err != nil {
		panic(err)
	}
}
```

### Sources
I basically just took code from [`uinput`](https://github.com/bendahl/uinput) and adapted it to my needs.