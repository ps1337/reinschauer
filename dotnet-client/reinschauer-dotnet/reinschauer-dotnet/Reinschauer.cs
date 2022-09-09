using System;
using System.Drawing;
using System.Drawing.Imaging;
using System.IO;
using System.Net;
using System.Runtime.InteropServices;
using System.Threading;
using System.Windows.Forms;
using Websocket.Client;

namespace reinschauer_dotnet
{
    public class Reinschauer
    {

        // SETTINGS
        private static long goodQuality = 20L;
        private static int scaler = 5;

        // NOT SETTINGS
        private static WebsocketClient client = null;
        private static System.Timers.Timer timer = new System.Timers.Timer();
        private static Bitmap bm, resized;
        private static Graphics g;
        private static MemoryStream memStream;
        private static ImageCodecInfo codecfInfo = null;

        private static EncoderParameters myEncoderParameters;
        private static EncoderParameter myEncoderParameter;
        private static System.Drawing.Imaging.Encoder myEncoder;

        // "mutex"
        private static bool already_running = false;

        public static void Main(string[] args)
        {
            if(args.Length != 3)
            {
                Console.WriteLine("Wrong number of arguments :/");
                Console.WriteLine("Usage: ./reinschauer-dotnet.exe <IP> <Port> <TLS Enabled (true/false)>");
                return;
            }
            Uri url;
            var exitEvent = new ManualResetEvent(false);
            if (args[2] == "true")
            {
                url = new Uri($"wss://{args[0]}:{args[1]}/messengerkeepalive");
            }
            else
            {
                url = new Uri($"ws://{args[0]}:{args[1]}/messengerkeepalive");
            }
            // yolo
            ServicePointManager.ServerCertificateValidationCallback += (sender, cert, chain, sslPolicyErrors) => { return true; };

            GetEncoderInfo("image/jpeg");
            if(codecfInfo == null)
            {
                Console.WriteLine("JPG codec not found, very bad, yo!");
            }

            // for all you crazy people with ultra wide screens
            if (Screen.PrimaryScreen.Bounds.Width > 4000)
            {
                scaler = 6;
            }

            myEncoderParameters = new EncoderParameters(1);
            myEncoder = System.Drawing.Imaging.Encoder.Quality;
            myEncoderParameter = new EncoderParameter(myEncoder, goodQuality);
            myEncoderParameters.Param[0] = myEncoderParameter;

            using (client = new WebsocketClient(url))
            {
                client.ReconnectTimeout = TimeSpan.FromSeconds(8);
                client.ReconnectionHappened.Subscribe(info =>
                    Console.WriteLine($"Reconnection happened, type: {info.Type}"));

                client.MessageReceived.Subscribe(msg => handleIncomingMessage(msg));
                client.Start();

                // 1 fps as default
                timer.Interval = 1000;
                timer.Elapsed += delegate {
                    if(already_running || client == null || !client.IsRunning) { return;  }
                    already_running = true;
                    bm = new Bitmap(Screen.PrimaryScreen.Bounds.Width, Screen.PrimaryScreen.Bounds.Height, PixelFormat.Format32bppArgb); 
                    g = Graphics.FromImage(bm);
                    g.CopyFromScreen(0, 0, 0, 0, bm.Size, CopyPixelOperation.SourceCopy);
                    resized = new Bitmap(bm, new Size(bm.Width / scaler, bm.Height / scaler));
                    memStream = new MemoryStream();

                    if(codecfInfo != null)
                    {
                        resized.Save(memStream, codecfInfo, myEncoderParameters);
                    }
                    else
                    {
                        resized.Save(memStream, ImageFormat.Jpeg);
                    }

                    if (client != null && client.IsRunning) {
                        client.Send(memStream.ToArray());
                    }
                    bm.Dispose();
                    resized.Dispose();
                    g.Dispose();
                    memStream.Dispose();
                    already_running = false;
                };
                timer.Enabled = true;

                exitEvent.WaitOne();
            }
        }

        // thanks MSDN
        private static void GetEncoderInfo(String mimeType)
        {
            int j;
            ImageCodecInfo[] encoders;
            encoders = ImageCodecInfo.GetImageEncoders();
            for (j = 0; j < encoders.Length; ++j)
            {
                if (encoders[j].MimeType == mimeType)
                {
                    codecfInfo = encoders[j];
                    break;
                }
            }
        }

        private static void handleIncomingMessage(Websocket.Client.ResponseMessage msg)
        {
            if(msg.MessageType != System.Net.WebSockets.WebSocketMessageType.Text) { return;  }
            //Console.WriteLine($"Message received: {msg}");
            var splitMessage = msg.ToString().Split(' ');
            string opcode = splitMessage[0];
            switch(opcode)
            {
                case "FPS":
                    if(splitMessage.Length == 2)
                    {
                        timer.Stop();
                        double fps = Convert.ToDouble(splitMessage[1]);
                        // it seems higher values are unstable across tunnels
                        if(fps > 10)
                        {
                            fps = 10;
                        }
                        timer.Interval = 1000 / fps;
                        timer.Start();
                    }
                    break;
                case "SCL":
                    if (splitMessage.Length == 2)
                    {
                        timer.Stop();
                        scaler = Convert.ToInt32(splitMessage[1]);
                        timer.Start();
                    }
                    break;
                case "QUL":
                    if (splitMessage.Length == 2)
                    {
                        timer.Stop();
                        goodQuality = Convert.ToInt64(splitMessage[1]);
                        myEncoderParameters = new EncoderParameters(1);
                        myEncoder = System.Drawing.Imaging.Encoder.Quality;
                        myEncoderParameter = new EncoderParameter(myEncoder, goodQuality);
                        myEncoderParameters.Param[0] = myEncoderParameter;
                        timer.Start();
                    }
                    break;
                case "LCL":
                    TriggerClick(msg.ToString(), false);
                    break;
                case "RCL":
                    TriggerClick(msg.ToString(), true);
                    break;
                case "KEY":
                    TriggerKey(msg.ToString());
                    break;
                case "ELO":
                    // ping
                    break;
                default:
                    break;
            }
        }

        // https://www.codeproject.com/Articles/5264831/How-to-Send-Inputs-using-Csharp
        [StructLayout(LayoutKind.Sequential)]
        public struct KeyboardInput
        {
            public ushort wVk;
            public ushort wScan;
            public uint dwFlags;
            public uint time;
            public IntPtr dwExtraInfo;
        }

        [StructLayout(LayoutKind.Sequential)]
        public struct MouseInput
        {
            public int dx;
            public int dy;
            public uint mouseData;
            public uint dwFlags;
            public uint time;
            public IntPtr dwExtraInfo;
        }

        [StructLayout(LayoutKind.Explicit)]
        public struct InputUnion
        {
            [FieldOffset(0)] public MouseInput mi;
            [FieldOffset(0)] public KeyboardInput ki;
        }

        public struct Input
        {
            public int type;
            public InputUnion u;
        }

        [Flags]
        public enum InputType
        {
            Mouse = 0,
            Keyboard = 1,
        }

        [Flags]
        public enum KeyEventF
        {
            KeyDown = 0x0000,
            ExtendedKey = 0x0001,
            KeyUp = 0x0002,
            Unicode = 0x0004,
            Scancode = 0x0008
        }

        [Flags]
        public enum MouseEventF
        {
            Absolute = 0x8000,
            HWheel = 0x01000,
            Move = 0x0001,
            MoveNoCoalesce = 0x2000,
            LeftDown = 0x0002,
            LeftUp = 0x0004,
            RightDown = 0x0008,
            RightUp = 0x0010,
            MiddleDown = 0x0020,
            MiddleUp = 0x0040,
            VirtualDesk = 0x4000,
            Wheel = 0x0800,
            XDown = 0x0080,
            XUp = 0x0100
        }

        [DllImport("user32.dll", SetLastError = true)]
        private static extern uint SendInput(uint nInputs, Input[] pInputs, int cbSize);
        [DllImport("user32.dll", SetLastError = true)]
        private static extern short VkKeyScan(char ch);


        private static void TriggerClick(string rawMessage, bool secondary)
        {
            var splitMessage = rawMessage.Split(' ');
            if(splitMessage.Length != 3) { return; }
            int tap_x = Convert.ToInt32(splitMessage[1]);
            int tap_y = Convert.ToInt32(splitMessage[2]);
            click(tap_x, tap_y, secondary);
        }

        private static void click(int x, int y, bool secondary)
        {
            uint flags = 0;
            if(secondary)
            {
                flags = (uint)(MouseEventF.Move | MouseEventF.RightDown | MouseEventF.RightUp | MouseEventF.Absolute);
            }
            else
            {
                flags = (uint)(MouseEventF.Move | MouseEventF.LeftDown | MouseEventF.LeftUp | MouseEventF.Absolute);

            }
            var input = new Input
            {
                type = (int)InputType.Mouse,
                u = new InputUnion
                {
                    mi = new MouseInput
                    {
                        dx = x,
                        dy = y,
                        dwFlags = flags
                    }
                }
            };
            Input[] inputs = { input };

            SendInput((uint)inputs.Length, inputs, Marshal.SizeOf(typeof(Input)));
        }

        private static void TriggerKey(string rawMessage) {
            Input input;
            int keycode = 0;
            int flags = 0;
            var splitMessage = rawMessage.Split(' ');
            if (splitMessage.Length < 2) { return; }

            if (splitMessage[1].Length == 1)
            {
                short keyscan = VkKeyScan(splitMessage[1][0]);
                keycode = keyscan & 0xff;
                flags = (keyscan >> 8) & 0xff;

                // shift down
                if(flags == 1) {
                    input = new Input
                    {
                        type = (int)InputType.Keyboard,
                        u = new InputUnion
                        {
                            ki = new KeyboardInput
                            {
                                wVk = 0xA0,
                                dwFlags = (uint)KeyEventF.KeyDown
                            }
                        }
                    };
                    Input[] inputs_sd = { input };
                    SendInput((uint)inputs_sd.Length, inputs_sd, Marshal.SizeOf(typeof(Input)));
                }
            }
            else
            {
                switch(splitMessage[1])
                {
                    case "Return":
                        keycode = 0x0D;
                        break;
                    case "BackSpace":
                        keycode = 0x08;
                        break;
                    case "Left":
                        keycode = 0x25;
                        break;
                    case "Up":
                        keycode = 0x26;
                        break;
                    case "Right":
                        keycode = 0x27;
                        break;
                    case "Down":
                        keycode = 0x28;
                        break;
                    case "LeftSuper":
                        keycode = 0x5B;
                        break;
                    case "RightSuper":
                        keycode = 0x5C;
                        break;
                    case "Escape":
                        keycode = 0x1B;
                        break;
                    case "Space":
                        keycode = 0x20;
                        break;
                    default:
                        break;
                }
            }

            // send the requested key event
            var inputDown = new Input
            {
                type = (int)InputType.Keyboard,
                u = new InputUnion
                {
                    ki = new KeyboardInput
                    {
                        wVk = (ushort)keycode,
                        dwFlags = (uint)KeyEventF.KeyDown
                    }
                }
            };
            var inputUp = new Input
            {
                type = (int)InputType.Keyboard,
                u = new InputUnion
                {
                    ki = new KeyboardInput
                    {
                        wVk = (ushort)keycode,
                        dwFlags = (uint)KeyEventF.KeyUp
                    }
                }
            };
            Input[] inputs = { inputDown, inputUp };
            SendInput((uint)inputs.Length, inputs, Marshal.SizeOf(typeof(Input)));

            // shift up
            if (flags == 1)
            {
                input = new Input
                {
                    type = (int)InputType.Keyboard,
                    u = new InputUnion
                    {
                        ki = new KeyboardInput
                        {
                            wVk = 0xA0,
                            dwFlags = (uint)KeyEventF.KeyUp
                        }
                    }
                };
                Input[] inputs_su = { input };
                SendInput((uint)inputs_su.Length, inputs_su, Marshal.SizeOf(typeof(Input)));
            }
        }

    }


}
