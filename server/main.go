package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/gorilla/websocket"
)

var banner = `
█▀█ █▀▀ █ █▄░█ █▀ █▀▀ █░█ ▄▀█ █░█ █▀▀ █▀█
█▀▄ ██▄ █ █░▀█ ▄█ █▄▄ █▀█ █▀█ █▄█ ██▄ █▀▄`

var DEFAULT_FPS = 10
var addr = flag.String("addr", "0.0.0.0:6969", "Listener")
var context = flag.String("context", "/messengerkeepalive", "Context Path (has to match with client)")
var server_key = flag.String("srvkey", "server.key", "Server Key")
var server_crt = flag.String("srvcrt", "server.crt", "Server Certificate")
var WINDOW fyne.Window
var derButton = newHackerButton(widget.NewButton("yolo", func() {}))
var upgrader = websocket.Upgrader{} // use default options
var CONN *websocket.Conn

// for absolute positioning in windows
var CONV_BASE = float32(65535)
var IMG = canvas.NewImageFromFile("reinschauer.jpg")
var slider = &widget.Slider{Step: 1, Min: 1, Max: 30, OnChanged: func(f float64) {
	if CONN == nil {
		return
	}
	CONN.WriteMessage(websocket.TextMessage, []byte("FPS "+strconv.Itoa(int(f))))
}}
var shiftHeld = false

func handler(w http.ResponseWriter, r *http.Request) {
	_conn, err := upgrader.Upgrade(w, r, nil)
	CONN = _conn
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer CONN.Close()

	box := container.NewPadded(IMG, derButton)
	slider_layout := container.NewHSplit(widget.NewLabel("FPS"), slider)
	slider_layout.SetOffset(0)
	split := container.NewVSplit(box, slider_layout)
	split.SetOffset(1)
	WINDOW.SetContent(split)

	// set default FPS
	fps_str := strconv.Itoa(DEFAULT_FPS)
	CONN.WriteMessage(websocket.TextMessage, []byte("FPS "+fps_str))
	slider.SetValue(float64(DEFAULT_FPS))

	for {
		messageType, message, err := CONN.ReadMessage()
		if err != nil {
			fmt.Println("Websocket Error: Error on ReadMessage()")
			fmt.Println(err.Error())
			continue
		}

		if messageType == websocket.BinaryMessage {
			// image payload
			jpgReader := bytes.NewReader(message)
			IMG = canvas.NewImageFromReader(jpgReader, "")
			IMG.FillMode = canvas.ImageFillStretch
			// update the image pointer
			box.Objects[0] = IMG

			box.Refresh()
		} else if messageType == websocket.TextMessage {
			// control message
		} else {
			// should never happen
			fmt.Println("Websocket Error: Unknown Message Type")
		}

	}
}

func startServer() error {
	flag.Parse()
	log.SetFlags(0)

	http.HandleFunc(*context, handler)
	return http.ListenAndServeTLS(*addr, *server_crt, *server_key, nil)
}

// a button with the Tappable interface implemented
// so we can get the local click coordinates
type HackerButton struct {
	widget.Button
	button *widget.Button
}

func (b *HackerButton) Tapped(e *fyne.PointEvent) {
	SendClick(false, e)
}

func (b *HackerButton) TappedSecondary(e *fyne.PointEvent) {
	SendClick(true, e)
}

func SendClick(secondary bool, e *fyne.PointEvent) {
	curr_img_size := IMG.Size()
	factor_x := CONV_BASE / curr_img_size.Width
	factor_y := CONV_BASE / curr_img_size.Height

	tap_x := int(e.Position.X * factor_x)
	tap_y := int(e.Position.Y * factor_y)
	if CONN == nil {
		fmt.Println("Websocket Error: Connection is nil")
		return
	} else if secondary {
		CONN.WriteMessage(websocket.TextMessage, []byte("RCL "+strconv.Itoa(tap_x)+" "+strconv.Itoa(tap_y)))
	} else {
		CONN.WriteMessage(websocket.TextMessage, []byte("LCL "+strconv.Itoa(tap_x)+" "+strconv.Itoa(tap_y)))
	}
}

func newHackerButton(b *widget.Button) *HackerButton {
	button := &HackerButton{button: b}
	button.ExtendBaseWidget(button)
	return button
}

func main() {

	fmt.Println(banner)

	myApp := app.New()
	WINDOW = myApp.NewWindow("R E I N S C H A U E R <3")

	if deskCanvas, ok := WINDOW.Canvas().(desktop.Canvas); ok {
		deskCanvas.SetOnKeyDown(func(key *fyne.KeyEvent) {
			if CONN == nil {
				return
			}
			toSend := ""
			if key.Name == "LeftShift" || key.Name == "RightShift" {
				shiftHeld = true
				return
			} else if len(key.Name) == 1 {
				if shiftHeld {
					toSend = strings.ToUpper(string(key.Name))
				} else {

					toSend = strings.ToLower(string(key.Name))
				}
			} else {
				// https://github.com/fyne-io/fyne/blob/master/key.go
				switch key.Name {
				case "Space":
					toSend = " "
				case "Return":
					toSend = "Return"
				case "BackSpace":
					toSend = "BackSpace"
				default:
					toSend = string(key.Name)
				}
			}
			CONN.WriteMessage(websocket.TextMessage, []byte("KEY "+toSend))
		})
		deskCanvas.SetOnKeyUp(func(key *fyne.KeyEvent) {
			if key.Name == "LeftShift" || key.Name == "RightShift" {
				shiftHeld = false
			}
		})
	}

	IMG.FillMode = canvas.ImageFillContain
	WINDOW.SetContent(IMG)

	go func() {
		for {
			err := startServer()
			if err != nil {
				fmt.Println(err.Error())
				fmt.Println("Restarting...")
			}
		}
	}()

	WINDOW.ShowAndRun()

}
