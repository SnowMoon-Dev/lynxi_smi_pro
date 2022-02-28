# lynxi-smi-pro
![GitHub](https://img.shields.io/github/license/SnowMoon-Dev/lynxi_smi_pro?label=license)
![Latest GitHub release](https://img.shields.io/github/release/SnowMoon-Dev/lynxi_smi_pro.svg)

Get informations about the system's APU(s), through lynxi-smi.

# Features
* Get the number of APUs 
* Get the driver's version
* Get the number of KA200
* Get APU or hardware Detail info.

# Usage
```
usage: lynxi-smi-pro [<flags>]

Optional flags:
  -h, --help             Show context-sensitive help (also try --help-long and --help-man).
  -q, --query            Display APU or hardware Detail info.
  -i, --index=INDEX      Target a specific Board index. This Flag is used to query APU or hardware Detail details
  -c, --chip_id=CHIP_ID  Target a specific Chip ID. This Flag is used to query APU or hardware Detail details
      --query-apu=name,driver_version,power,...
                         Query Information about APU.
  -L, --list-apus        Display a list of APUs connected to the system.
      --chip-count       Displays the number of KA200.
      --chip-list        Displays a list of KA200.
      --debug            Display Debug Info
      --help-query-apu   Display Help Query Information about APU.
```
