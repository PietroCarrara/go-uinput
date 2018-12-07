package uinput

import (
	"fmt"
	"io"
	"os"
	"time"
	"unsafe"
)

// Joystick interface
type Joystick interface {
	BtnDown(btn uint16) error
	BtnUp(btn uint16) error

	LeftStickX(x int32) error
	LeftStickY(y int32) error
	RightStickX(x int32) error
	RightStickY(y int32) error

	io.Closer
}

type vJoystick struct {
	devFile *os.File
}

func setupJoystick(devFile *os.File, minX int32, maxX int32, minY int32, maxY int32, flat int32, fuzz int32) error {
	var uinp uinputUserDev

	buttons := []uint16{
		BtnSouth,
		BtnEast,
		BtnNorth,
		BtnWest,
		BtnTL,
		BtnTR,
		BtnTL2,
		BtnTR2,
		BtnSelect,
		BtnStart,
		BtnMode,
		BtnThumbL,
		BtnThumbR,

		BtnDpadUp,
		BtnDpadDown,
		BtnDpadLeft,
		BtnDpadRight,
	}

	hats := []uint16{
		AbsHat0X,
		AbsHat0Y,
	}

	// TODO: add possibility to change these values
	uinp.Name = uinputSetupNameToBytes([]byte("GoUinputDevice"))
	uinp.ID.BusType = BusVirtual
	uinp.ID.Vendor = 1
	uinp.ID.Product = 2
	uinp.ID.Version = 3

	// Sticks
	uinp.AbsMin[AbsX] = minX
	uinp.AbsMax[AbsX] = maxX
	uinp.AbsFuzz[AbsX] = fuzz
	uinp.AbsFlat[AbsX] = flat

	uinp.AbsMin[AbsY] = minY
	uinp.AbsMax[AbsY] = maxY
	uinp.AbsFuzz[AbsY] = fuzz
	uinp.AbsFlat[AbsY] = flat

	uinp.AbsMin[AbsRX] = minX
	uinp.AbsMax[AbsRX] = maxX
	uinp.AbsFuzz[AbsRX] = fuzz
	uinp.AbsFlat[AbsRX] = flat

	uinp.AbsMin[AbsRY] = minY
	uinp.AbsMax[AbsRY] = maxY
	uinp.AbsFuzz[AbsRY] = fuzz
	uinp.AbsFlat[AbsRY] = flat

	// Digital dpad buttons
	for _, i := range hats {
		uinp.AbsMax[i] = 1
		uinp.AbsMin[i] = -1
	}

	buf, err := uinputUserDevToBuffer(uinp)
	if err != nil {
		goto err
	}

	err = ioctl(devFile, uiSetEvBit, EvKey)
	if err != nil {
		err = fmt.Errorf("Could not perform UI_SET_EVBIT ioctl: %v", err)
		goto err
	}

	// Register all gamepad buttons
	for _, i := range buttons {
		err = ioctl(devFile, uiSetKeyBit, uintptr(i))
		if err != nil {
			err = fmt.Errorf("Could not perform UI_SET_KEYBIT ioctl: %v", err)
			goto err
		}
	}

	// Configure the sticks (and the digital dpad buttons)
	err = ioctl(devFile, uiSetEvBit, EvAbs)
	if err != nil {
		err = fmt.Errorf("Could not perform UI_SET_EVBIT ioctl: %v", err)
		goto err
	}

	err = ioctl(devFile, uiSetAbsBit, uintptr(AbsX))
	if err != nil {
		err = fmt.Errorf("Could not perform UI_SET_EVBIT ioctl: %v", err)
		goto err
	}

	err = ioctl(devFile, uiSetAbsBit, uintptr(AbsY))
	if err != nil {
		err = fmt.Errorf("Could not perform UI_SET_EVBIT ioctl: %v", err)
		goto err
	}

	err = ioctl(devFile, uiSetAbsBit, uintptr(AbsRX))
	if err != nil {
		err = fmt.Errorf("Could not perform UI_SET_EVBIT ioctl: %v", err)
		goto err
	}

	err = ioctl(devFile, uiSetAbsBit, uintptr(AbsRY))
	if err != nil {
		err = fmt.Errorf("Could not perform UI_SET_EVBIT ioctl: %v", err)
		goto err
	}

	// Dpad buttons
	for _, i := range hats {
		err = ioctl(devFile, uiSetAbsBit, uintptr(i))
		if err != nil {
			err = fmt.Errorf("Could not perform UI_SET_EVBIT ioctl: %v", err)
			goto err
		}
	}

	err = ioctl(devFile, uiDevSetup, uintptr(unsafe.Pointer(&uinp)))
	if err != nil {
		err = fmt.Errorf("Could not perform UI_DEV_SETUP ioctl: %v", err)
		goto err
	}

	_, err = devFile.Write(buf)
	if err != nil {
		err = fmt.Errorf("Could not write uinputUserDev to device: %v", err)
		goto err
	}

	err = ioctl(devFile, uiDevCreate, uintptr(0))
	if err != nil {
		devFile.Close()
		err = fmt.Errorf("Could not perform UI_DEV_CREATE ioctl: %v", err)
		goto err
	}

	time.Sleep(time.Millisecond * 200)

	return nil

err:
	return err
}

func emitBtnDown(devFile *os.File, code uint16) error {
	err := emitEvent(devFile, EvKey, code, 1)
	if err != nil {
		return fmt.Errorf("Could not emit key down event: %v", err)
	}

	err = emitEvent(devFile, EvSyn, SynReport, 0)
	if err != nil {
		return fmt.Errorf("Could not emit sync event: %v", err)
	}

	return err
}

func emitBtnUp(devFile *os.File, code uint16) error {
	err := emitEvent(devFile, EvKey, code, 0)
	if err != nil {
		return fmt.Errorf("Could not emit key up event: %v", err)
	}

	err = emitEvent(devFile, EvSyn, SynReport, 0)
	if err != nil {
		return fmt.Errorf("Could not emit sync event: %v", err)
	}

	return err
}

// CreateJoystick creates a virtual input device that emulates a joystick
func CreateJoystick(minX int32, maxX int32, minY int32, maxY int32, flat int32, fuzz int32) (Joystick, error) {
	dev, err := openUinputDev()
	if err != nil {
		return nil, err
	}

	err = setupJoystick(dev, minX, maxX, minY, maxY, flat, fuzz)
	if err != nil {
		return nil, err
	}

	return vJoystick{devFile: dev}, err
}

// BtnDown presses and holds a button
func (vj vJoystick) BtnDown(btn uint16) error {

	if btn == BtnDpadUp || btn == BtnDpadDown || btn == BtnDpadLeft || btn == BtnDpadRight {
		vj.dpadDown(btn)
	}

	err := emitEvent(vj.devFile, EvKey, btn, 1)
	if err != nil {
		return fmt.Errorf("Could not emit dpad down event: %v", err)
	}

	err = emitEvent(vj.devFile, EvSyn, SynReport, 0)
	if err != nil {
		return fmt.Errorf("Could not emit sync event: %v", err)
	}

	return err
}

func (vj vJoystick) dpadDown(btnCode uint16) error {

	var hat uint16
	var val int32

	switch btnCode {
	case BtnDpadUp:
		hat = AbsHat0Y
		val = -1
	case BtnDpadDown:
		hat = AbsHat0Y
		val = 1
	case BtnDpadLeft:
		hat = AbsHat0X
		val = -1
	case BtnDpadRight:
		hat = AbsHat0X
		val = 1
	default:
		return fmt.Errorf("Unknown dpad button: %x", btnCode)
	}

	err := emitEvent(vj.devFile, EvAbs, hat, val)
	if err != nil {
		return fmt.Errorf("Could not emit button up event: %v", err)
	}

	err = emitEvent(vj.devFile, EvSyn, SynReport, 0)
	if err != nil {
		return fmt.Errorf("Could not emit sync event: %v", err)
	}

	return nil
}

// BtnUp releases a button
func (vj vJoystick) BtnUp(btn uint16) error {
	if btn == BtnDpadUp || btn == BtnDpadDown || btn == BtnDpadLeft || btn == BtnDpadRight {
		vj.dpadUp(btn)
	}

	err := emitEvent(vj.devFile, EvKey, btn, 0)
	if err != nil {
		return fmt.Errorf("Could not emit button up event: %v", err)
	}

	err = emitEvent(vj.devFile, EvSyn, SynReport, 0)
	if err != nil {
		return fmt.Errorf("Could not emit sync event: %v", err)
	}

	return err
}

func (vj vJoystick) dpadUp(btnCode uint16) error {

	var hat uint16

	switch btnCode {
	case BtnDpadUp, BtnDpadDown:
		hat = AbsHat0Y
	case BtnDpadLeft, BtnDpadRight:
		hat = AbsHat0X
	default:
		return fmt.Errorf("Unknown dpad button: %x", btnCode)
	}

	err := emitEvent(vj.devFile, EvAbs, hat, 0)
	if err != nil {
		return fmt.Errorf("Could not emit dpad up event: %v", err)
	}

	err = emitEvent(vj.devFile, EvSyn, SynReport, 0)
	if err != nil {
		return fmt.Errorf("Could not emit sync event: %v", err)
	}

	return nil
}

func (vj vJoystick) LeftStickX(x int32) error {
	err := emitEvent(vj.devFile, EvAbs, AbsX, x)
	if err != nil {
		return fmt.Errorf("Could not emit AbsX event: %v", err)
	}

	err = emitEvent(vj.devFile, EvSyn, SynReport, 0)
	if err != nil {
		return fmt.Errorf("Could not emit sync event: %v", err)
	}

	return nil
}

func (vj vJoystick) LeftStickY(y int32) error {
	err := emitEvent(vj.devFile, EvAbs, AbsY, y)
	if err != nil {
		return fmt.Errorf("Could not emit AbsY event: %v", err)
	}

	err = emitEvent(vj.devFile, EvSyn, SynReport, 0)
	if err != nil {
		return fmt.Errorf("Could not emit sync event: %v", err)
	}

	return nil
}

func (vj vJoystick) RightStickY(y int32) error {
	err := emitEvent(vj.devFile, EvAbs, AbsRY, y)
	if err != nil {
		return fmt.Errorf("Could not emit AbsRY event: %v", err)
	}

	err = emitEvent(vj.devFile, EvSyn, SynReport, 0)
	if err != nil {
		return fmt.Errorf("Could not emit sync event: %v", err)
	}

	return nil
}

func (vj vJoystick) RightStickX(x int32) error {
	err := emitEvent(vj.devFile, EvAbs, AbsRX, x)
	if err != nil {
		return fmt.Errorf("Could not emit AbsRX event: %v", err)
	}

	err = emitEvent(vj.devFile, EvSyn, SynReport, 0)
	if err != nil {
		return fmt.Errorf("Could not emit sync event: %v", err)
	}

	return nil
}

func (vj vJoystick) Close() error {
	return destroyDevice(vj.devFile)
}
