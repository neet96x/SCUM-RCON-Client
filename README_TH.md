# rcon.exe Quick Help

Ref: [SCUM-RCON Server releases](https://github.com/herbie96x/SCUM-RCON/releases)

## Config

ไฟล์ config อยู่ที่:

```text
ini\scum_rcon.ini
```

```ini
[server]
host = 127.0.0.1
port = 9010
password = YOUR_RCON_PASSWORD
```

## ใช้งานเร็ว

ส่งคำสั่งเดียว:

```powershell
.\rcon.exe -commands "ListSpawnedVehicles"
.\rcon.exe -commands "CheckServerTime"
.\rcon.exe -commands "SpawnItem Weapon_M82A1 1 Location <steam64>"
```

## หลายคำสั่งในบรรทัดเดียว

```powershell
.\rcon.exe "ListPlayers;ListSpawnedVehicles;CheckServerTime;SpawnItem Weapon_M82A1 1 Location <steam64>"
```

## ไม่ใช้ไฟล์ config

ระบุ host, port, password ผ่าน command line:

```powershell
.\rcon.exe -host "127.0.0.1" -port 9010 -password "passwd" -commands "ListPlayers"
.\rcon.exe -host "127.0.0.1" -port 9010 -password "passwd" -commands "ListPlayers;ListSpawnedVehicles;CheckServerTime;SpawnItem Weapon_M82A1 1 Location <steam64>"
```

## หมายเหตุ

- ถ้าใช้หลายคำสั่ง ให้คั่นด้วย `;`
- ถ้าใช้ `;` ใน PowerShell ต้องครอบทั้งชุดด้วย quote
