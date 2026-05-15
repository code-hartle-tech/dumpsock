# Supported devices

DumpSock uses Apple's standard USB pairing + AFC service. Anything Finder can show you under "Files on This iPhone" is fair game.

## Tested

| Device | iOS | Status |
|---|---|---|
| iPhone 16 Pro Max | 26.4.2 | ✅ Daily driver (operator's own device) |
| iPad Pro (M2 / M4) | 18.x | ⚙ Expected to work — same protocol |

## Should-just-work

- **Any iPhone** running iOS 13+. The AFC protocol is stable across that range.
- **Any iPad** running iPadOS 13+.
- **iPod touch (7th gen)** running iOS 13–15.

If your device pairs with macOS Finder, it pairs with DumpSock.

## Old iPhones

Pre-iOS 13 iPhones still work in principle — older AFC versions are protocol-compatible. We don't test them. If you have one and it works (or doesn't), let us know.

## Apple Watch

No. Apple Watch USB is for Apple internal use only and the diagnostic port is hidden.

## Android / other phones

No. DumpSock is iOS-only by design. For Android, `mtpfs` or `gphoto2` work natively over USB MTP/PTP. Or back up to your computer via Google Drive and pull from there.

## Multi-device setup

If you have multiple iPhones plugged in at once, DumpSock picks the first one by default. Use `--udid` to specify which:

```bash
dumpsock devices              # list everything attached
dumpsock pull --udid 00008140-001804A62209801C
```
