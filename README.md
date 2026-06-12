# rcon.exe Quick Help

Ref: [SCUM-RCON Server releases](https://github.com/herbie96x/SCUM-RCON/releases)

## Config

Config file:

```text
ini\scum_rcon.ini
```

```ini
[server]
host = 192.168.50.3
port = 9010
password = YOUR_RCON_PASSWORD
```

## Use

```powershell
.\rcon.exe -commands "ListSpawnedVehicles"
.\rcon.exe -commands "SpawnItem Weapon_M82A1 1 Location <steam64>"
.\rcon.exe "ListPlayers;ListSpawnedVehicles;CheckServerTime;SpawnItem Weapon_M82A1 1 Location <steam64>"
.\rcon.exe -host "127.0.0.1" -port 9010 -password "passwd" -commands "ListPlayers;ListSpawnedVehicles;CheckServerTime;SpawnItem Weapon_M82A1 1 Location <steam64>"
```

## Notes

- Separate multiple commands with `;`
- Quote the whole command list in PowerShell
