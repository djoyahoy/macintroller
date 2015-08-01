package main

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Cocoa
// #import <Foundation/Foundation.h>
// #import <CoreGraphics/CoreGraphics.h>
import "C"

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/veandco/go-sdl2/sdl"
)

type config struct {
	Axes     map[string][3]int `json:"axes"`
	Buttons  map[string]int    `json:"buttons"`
	Triggers map[string][2]int `json:"triggers"`
}

func newConfig() config {
	return config{
		Axes:     make(map[string][3]int),
		Buttons:  make(map[string]int),
		Triggers: make(map[string][2]int),
	}
}

func createKey(val int) (C.CGEventRef, C.CGEventRef, error) {
	up := C.CGEventCreateKeyboardEvent(nil, C.CGKeyCode(val), false)
	if up == nil {
		return nil, nil, fmt.Errorf("unable to create key %d", val)
	}

	down := C.CGEventCreateKeyboardEvent(nil, C.CGKeyCode(val), true)
	if down == nil {
		return nil, nil, fmt.Errorf("unable to create key %d", val)
	}

	return up, down, nil
}

/**************************************************
	Axis
**************************************************/
type axis struct {
	threshold int
	negative  bool
	positive  bool
	nUp       C.CGEventRef
	nDown     C.CGEventRef
	pUp       C.CGEventRef
	pDown     C.CGEventRef
}

func openAxis(threshold int, neg int, pos int) (*axis, error) {
	nUp, nDown, err := createKey(neg)
	if err != nil {
		return nil, err
	}

	pUp, pDown, err := createKey(pos)
	if err != nil {
		return nil, err
	}

	a := &axis{
		threshold: threshold,
		negative:  false,
		positive:  false,
		nUp:       nUp,
		nDown:     nDown,
		pUp:       pUp,
		pDown:     pDown,
	}
	return a, nil
}

func (a *axis) close() {
	C.CFRelease(C.CFTypeRef(a.nUp))
	C.CFRelease(C.CFTypeRef(a.nDown))
	C.CFRelease(C.CFTypeRef(a.pUp))
	C.CFRelease(C.CFTypeRef(a.pDown))
}

func (a *axis) handleEvent(val int) {
	if val <= -a.threshold && !a.negative {
		C.CGEventPost(C.kCGAnnotatedSessionEventTap, a.nDown)
		a.negative = true
	} else if val > -a.threshold && a.negative {
		C.CGEventPost(C.kCGAnnotatedSessionEventTap, a.nUp)
		a.negative = false
	}

	if val >= a.threshold && !a.positive {
		C.CGEventPost(C.kCGAnnotatedSessionEventTap, a.pDown)
		a.positive = true
	} else if val < a.threshold && a.positive {
		C.CGEventPost(C.kCGAnnotatedSessionEventTap, a.pUp)
		a.positive = false
	}
}

/***********************************************
	Button
***********************************************/
type button struct {
	up   C.CGEventRef
	down C.CGEventRef
}

func openButton(val int) (*button, error) {
	up, down, err := createKey(val)
	if err != nil {
		return nil, err
	}
	return &button{up: up, down: down}, nil
}

func (b *button) close() {
	C.CFRelease(C.CFTypeRef(b.up))
	C.CFRelease(C.CFTypeRef(b.down))
}

func (b *button) handleUp() {
	C.CGEventPost(C.kCGAnnotatedSessionEventTap, b.up)
}

func (b *button) handleDown() {
	C.CGEventPost(C.kCGAnnotatedSessionEventTap, b.down)
}

/****************************************
	Trigger
****************************************/
type trigger struct {
	threshold int
	active    bool
	up        C.CGEventRef
	down      C.CGEventRef
}

func openTrigger(threshold int, val int) (*trigger, error) {
	up, down, err := createKey(val)
	if err != nil {
		return nil, err
	}
	t := &trigger{
		threshold: threshold,
		active:    false,
		up:        up,
		down:      down,
	}
	return t, nil
}

func (t *trigger) handleEvent(val int) {
	if val >= t.threshold && !t.active {
		C.CGEventPost(C.kCGAnnotatedSessionEventTap, t.down)
		t.active = true
	} else if val < t.threshold && t.active {
		C.CGEventPost(C.kCGAnnotatedSessionEventTap, t.up)
		t.active = false
	}
}

func (t *trigger) close() {
	C.CFRelease(C.CFTypeRef(t.up))
	C.CFRelease(C.CFTypeRef(t.down))
}

/************************************************
	Controller
************************************************/
type controller struct {
	Axes     map[int]*axis
	Buttons  map[int]*button
	Triggers map[int]*trigger
}

func newController(conf config) (*controller, error) {
	axes := make(map[int]*axis)
	for k, v := range conf.Axes {
		a, err := openAxis(v[0], v[1], v[2])
		if err != nil {
			return nil, err
		}

		i, err := strconv.Atoi(k)
		if err != nil {
			return nil, err
		}

		axes[i] = a
	}

	buttons := make(map[int]*button)
	for k, v := range conf.Buttons {
		b, err := openButton(v)
		if err != nil {
			return nil, err
		}

		i, err := strconv.Atoi(k)
		if err != nil {
			return nil, err
		}

		buttons[i] = b
	}

	triggers := make(map[int]*trigger)
	for k, v := range conf.Triggers {
		t, err := openTrigger(v[0], v[1])
		if err != nil {
			return nil, err
		}

		i, err := strconv.Atoi(k)
		if err != nil {
			return nil, err
		}

		triggers[i] = t
	}

	return &controller{Axes: axes, Buttons: buttons, Triggers: triggers}, nil
}

func (c *controller) close() {
	for _, v := range c.Axes {
		v.close()
	}

	for _, v := range c.Buttons {
		v.close()
	}

	for _, v := range c.Triggers {
		v.close()
	}
}

func main() {
	defaultConfig := config{
		Axes: map[string][3]int{
			strconv.Itoa(sdl.CONTROLLER_AXIS_LEFTX):  {0x4000, 0x7B, 0x7C}, // Arrow Left + Right
			strconv.Itoa(sdl.CONTROLLER_AXIS_LEFTY):  {0x4000, 0x7E, 0x7D}, // Arrow Up + Down
			strconv.Itoa(sdl.CONTROLLER_AXIS_RIGHTX): {0x4000, 0x00, 0x07}, // A + X
			strconv.Itoa(sdl.CONTROLLER_AXIS_RIGHTY): {0x4000, 0x01, 0x06}, // S + Z
		},
		Buttons: map[string]int{
			strconv.Itoa(sdl.CONTROLLER_BUTTON_DPAD_UP):       0X7E, // Up
			strconv.Itoa(sdl.CONTROLLER_BUTTON_DPAD_DOWN):     0X7D, // Down
			strconv.Itoa(sdl.CONTROLLER_BUTTON_DPAD_LEFT):     0x7B, // Left
			strconv.Itoa(sdl.CONTROLLER_BUTTON_DPAD_RIGHT):    0x7C, // Right
			strconv.Itoa(sdl.CONTROLLER_BUTTON_A):             0X06, // Z
			strconv.Itoa(sdl.CONTROLLER_BUTTON_B):             0x07, // X
			strconv.Itoa(sdl.CONTROLLER_BUTTON_X):             0x00, // A
			strconv.Itoa(sdl.CONTROLLER_BUTTON_Y):             0x01, // S
			strconv.Itoa(sdl.CONTROLLER_BUTTON_LEFTSHOULDER):  0x0C, // Q
			strconv.Itoa(sdl.CONTROLLER_BUTTON_RIGHTSHOULDER): 0x0D, // W
			strconv.Itoa(sdl.CONTROLLER_BUTTON_LEFTSTICK):     0x38, // Shift
			strconv.Itoa(sdl.CONTROLLER_BUTTON_RIGHTSTICK):    0x3B, // Ctrl
			strconv.Itoa(sdl.CONTROLLER_BUTTON_BACK):          0x33, // Delete
			strconv.Itoa(sdl.CONTROLLER_BUTTON_GUIDE):         0x30, // Tab
			strconv.Itoa(sdl.CONTROLLER_BUTTON_START):         0x24, // Return
		},
		Triggers: map[string][2]int{
			strconv.Itoa(sdl.CONTROLLER_AXIS_TRIGGERLEFT):  {0x4000, 0x0E}, // E
			strconv.Itoa(sdl.CONTROLLER_AXIS_TRIGGERRIGHT): {0x4000, 0x0F}, // R
		},
	}

	conf := newConfig()
	if file, err := os.Open("config.json"); err != nil {
		f, err := os.Create("config.json")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		conf = defaultConfig

		buf, err := json.MarshalIndent(conf, "", "    ")
		if err != nil {
			log.Fatal(err)
		}
		f.Write(buf)
	} else {
		err := json.NewDecoder(file).Decode(&conf)
		if err != nil {
			os.Remove("config.json")
			log.Fatal(err)
		}

		file.Close()
	}

	controller, err := newController(conf)
	if err != nil {
		log.Fatal(err)
	}
	defer controller.close()

	err = sdl.Init((1 << 14) | sdl.INIT_GAMECONTROLLER)
	if err != nil {
		log.Fatal(err)
	}
	defer sdl.Quit()

	sdlController := sdl.GameControllerOpen(0)
	if sdlController == nil {
		log.Fatal(sdl.GetError())
	}
	defer sdlController.Close()

	for {
		for e := sdl.PollEvent(); e != nil; e = sdl.PollEvent() {
			switch e := e.(type) {
			case *sdl.ControllerButtonEvent:
				if e.State == 0 {
					controller.Buttons[int(e.Button)].handleUp()
				} else {
					controller.Buttons[int(e.Button)].handleDown()
				}
			case *sdl.ControllerAxisEvent:
				if e.Axis == sdl.CONTROLLER_AXIS_TRIGGERLEFT || e.Axis == sdl.CONTROLLER_AXIS_TRIGGERRIGHT {
					controller.Triggers[int(e.Axis)].handleEvent(int(e.Value))
				} else {
					controller.Axes[int(e.Axis)].handleEvent(int(e.Value))
				}
			}
		}
		sdl.Delay(10)
	}
}
