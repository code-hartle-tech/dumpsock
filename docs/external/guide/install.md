# Install

## macOS

**Recommended (GUI):**

1. Grab the latest `.app` from [GitHub Releases](https://github.com/code-hartle-tech/dumpsock/releases).
2. Drag `DumpSock.app` to `/Applications/`.
3. Open it. macOS may ask you to confirm the unsigned developer once (Right-click → Open). Code-signing arrives in [issue #5](https://github.com/code-hartle-tech/dumpsock/issues/5).

**CLI only:**

```bash
# pick the right arch for your Mac
curl -L -o dumpsock https://github.com/code-hartle-tech/dumpsock/releases/latest/download/dumpsock-darwin-arm64
chmod +x dumpsock
./dumpsock --help
```

## Linux

```bash
curl -L -o dumpsock https://github.com/code-hartle-tech/dumpsock/releases/latest/download/dumpsock-linux-amd64
chmod +x dumpsock
./dumpsock --help
```

`usbmuxd` must be installed and running. On Debian/Ubuntu: `sudo apt install usbmuxd libimobiledevice6`. On Fedora: `sudo dnf install usbmuxd libimobiledevice`.

## Windows

```powershell
# In PowerShell
Invoke-WebRequest -Uri "https://github.com/code-hartle-tech/dumpsock/releases/latest/download/dumpsock-windows-amd64.exe" -OutFile "dumpsock.exe"
.\dumpsock.exe --help
```

Apple Mobile Device Support must be installed (it ships with iTunes / Apple Devices on Windows).

## Build from source

```bash
git clone https://github.com/code-hartle-tech/dumpsock
cd dumpsock
bash scripts/build.sh         # all platforms
bash scripts/build-gui.sh     # macOS .app (macOS only)
```

Requires Go 1.24+. The macOS GUI build also needs Xcode Command Line Tools.

## Pair the iPhone first

DumpSock relies on Apple's standard USB pairing. The very first time you connect your iPhone to your Mac:

1. Plug it in.
2. Unlock the iPhone.
3. On the iPhone, tap **Trust This Computer**.
4. Enter your iPhone passcode.

DumpSock does not handle this — Finder / Settings does. After the first time, the pairing record is cached on both sides and DumpSock sees the phone instantly on every subsequent plug-in.
