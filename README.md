# rcon.exe Quick Help

Ref: [SCUM-RCON Server releases](https://github.com/herbie96x/SCUM-RCON/releases)

## Use  No folder or .ini file required for this command. 

```powershell
.\rcon.exe --host "127.0.0.1" --port 9010 --password "passwd"  -commands "ListPlayers;ListSpawnedVehicles;CheckServerTime"
```
## Config

Config file:

```text
ini\scum_rcon.ini
```

```ini
[server]
host = 127.0.0.1
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
