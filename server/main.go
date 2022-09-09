package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

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

var DEFAULT_FPS = 1
var DEFAULT_SCALER = 2
var DEFAULT_QUALITY = 20

var addr = flag.String("addr", "0.0.0.0:6969", "Listener")
var context = flag.String("context", "/messengerkeepalive", "Context Path (has to match with client)")
var server_key = flag.String("srvkey", "server.key", "Server Key")
var server_crt = flag.String("srvcrt", "server.crt", "Server Certificate")
var no_tls = flag.Bool("noTLS", false, "Don't use TLS")
var WINDOW fyne.Window
var derButton = newHackerButton(widget.NewButton("yolo", func() {}))
var upgrader = websocket.Upgrader{} // use default options
var CONN *websocket.Conn

// for absolute positioning in windows
var CONV_BASE = float32(65535)
var IMG = canvas.NewImageFromFile("reinschauer.jpg")
var fps_slider = &widget.Slider{Step: 1, Min: 1, Max: 30, OnChanged: func(f float64) {
	if CONN == nil {
		return
	}
	CONN.WriteMessage(websocket.TextMessage, []byte("FPS "+strconv.Itoa(int(f))))
}}

var scaler_slider = &widget.Slider{Step: 1, Min: 1, Max: 10, OnChanged: func(f float64) {
	if CONN == nil {
		return
	}
	CONN.WriteMessage(websocket.TextMessage, []byte("SCL "+strconv.Itoa(int(f))))
}}

var quality_slider = &widget.Slider{Step: 10, Min: 1, Max: 100, OnChanged: func(f float64) {
	if CONN == nil {
		return
	}
	CONN.WriteMessage(websocket.TextMessage, []byte("QUL "+strconv.Itoa(int(f))))
}}

func handler(w http.ResponseWriter, r *http.Request) {
	_conn, err := upgrader.Upgrade(w, r, nil)
	CONN = _conn
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer CONN.Close()

	box := container.NewPadded(IMG, derButton)

	fps_slider_layout := container.NewHSplit(widget.NewLabel("FPS"), fps_slider)
	fps_slider_layout.SetOffset(0)

	scaler_slider_layout := container.NewHSplit(widget.NewLabel("Scaler"), scaler_slider)
	scaler_slider_layout.SetOffset(0)

	quality_slider_layout := container.NewHSplit(widget.NewLabel("Quality"), quality_slider)
	quality_slider_layout.SetOffset(0)

	settingsBox := container.NewVBox(fps_slider_layout, scaler_slider_layout, quality_slider_layout)

	split := container.NewVSplit(box, settingsBox)
	split.SetOffset(1)
	WINDOW.SetContent(split)

	// set default values
	fps_str := strconv.Itoa(DEFAULT_FPS)
	CONN.WriteMessage(websocket.TextMessage, []byte("FPS "+fps_str))
	fps_slider.SetValue(float64(DEFAULT_FPS))

	scaler_str := strconv.Itoa(DEFAULT_SCALER)
	CONN.WriteMessage(websocket.TextMessage, []byte("SCL "+scaler_str))
	scaler_slider.SetValue(float64(DEFAULT_SCALER))

	quality_str := strconv.Itoa(DEFAULT_QUALITY)
	CONN.WriteMessage(websocket.TextMessage, []byte("QUL "+quality_str))
	quality_slider.SetValue(float64(DEFAULT_QUALITY))

	// send regular pings
	go func() {
		for {
			if CONN == nil {
				continue
			}
			CONN.WriteMessage(websocket.TextMessage, []byte("ELO"))
			time.Sleep(time.Second * 3)
		}
	}()

	for {
		messageType, message, err := CONN.ReadMessage()
		if err != nil {
			fmt.Println("Websocket Error: Error on ReadMessage()")
			fmt.Println(err.Error())
			continue
		}

		// image payload
		if messageType == websocket.BinaryMessage {
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
	if *no_tls {
		fmt.Println("Not Using TLS")
		return http.ListenAndServe(*addr, nil)
	} else {
		fmt.Println("Using TLS")
		return http.ListenAndServeTLS(*addr, *server_crt, *server_key, nil)
	}
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

	WINDOW.Canvas().SetOnTypedRune(func(r rune) {
		if CONN == nil {
			return
		}
		CONN.WriteMessage(websocket.TextMessage, []byte("KEY "+string(r)))
	})

	if deskCanvas, ok := WINDOW.Canvas().(desktop.Canvas); ok {
		deskCanvas.SetOnKeyDown(func(key *fyne.KeyEvent) {
			if CONN == nil {
				return
			}

			// single char -> handled via SetOnTypedRune
			if len(key.Name) == 1 {
				return
			}
			CONN.WriteMessage(websocket.TextMessage, []byte("KEY "+string(key.Name)))
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
