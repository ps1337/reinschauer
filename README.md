# Reinschauer

![Hello](https://github.com/ps1337/reinschauer/blob/main/server/reinschauer.jpg?raw=true)

A PoC to remotely control Windows machines over Websockets.

![Hello](https://github.com/ps1337/reinschauer/blob/main/res/reinschauer.gif?raw=true)

- Other than most HVNC implementations, `reinschauer` converts raw bitmaps to JPEG before sending data across the network to reduce frame size.
- FPS can be set via the GUI.
- Basic mouse and keyboard controls are possible.
- You can use the script in the `server` folder to generate TLS server files or bring your own.
- The server window can be resized freely, while click events should™ be translated to the correct pixel on the target machine.
## Protocol

```
+----------------------------------------------------------------------------------------+
|                                                                                        |
|                                                                                        |
|                  +----------------------------------------------------+                |
|                  |#1: Type (Binary or Text)                           |                |
|                  |    Binary: JPG Frame                               |                |
|                  |                                                    |                |
|                  |#2: Text                                            |                |
|                  |    FPS <FPS Count>                                 |                |
|                  |    LCL X Y (Left Click + Coordinates)              |                |
|                  |    RCL X Y                                         |                |
|                  |    KEY <Char>                                      |      xxxxxx    |
|       xxxxx   <--+----------------------------------------------------+--> xxxx   xx   |
|       x   xx                     Websockets via TLS                        xx       x  |
| x     xxxxxx                                                                xxxxxxxxx  |
| xxx     x   xx                                                          xx    x        |
|     xxxxxxxx                                                              xxxxx xxxxxx |
|         x                                                                   xxxxx      |
|         xx                                                                   xx        |
|         x xx                                                                 xxx       |
|        xx  xx                                                                x xx      |
|        x    xx                                                              xx  xxx    |
|       x      xx                                                             x     xx   |
|       x       x                                                            xx      x   |
+----------------------------------------------------------------------------------------+
```

## FAQ

**How to build?**

- Install a new version of golang, `>=1.18`
- For debian-based distros: `sudo apt -y install libx11-dev libxcursor-dev xorg-dev libgl1-mesa-dev`

```bash
$ cd client && GOOS=windows GOARCH=amd64 go build
$ cd server && GOOS=linux GOARCH=amd64 go build
```

> You may have to `go get` stuff before. Use `-ldflags -H=windowsgui` to disable to console window.

**Who's the client/server?**

> The `client` is executed on the target (Windows) machine. The `server` component is executed on the tester's (Linux) machine.

**Is this a HVNC / Hidden Desktop?**

> No. It uses the same Desktop as the user.

**Some keys and key combinations do not work, pls fix**

> I know. I have better things to do :P

**The Client uses too much CPU Time**

> Using a lower FPS value may work.

**I don't have a direct connection between `client` and `server`**

> Use a `socat` redirector.