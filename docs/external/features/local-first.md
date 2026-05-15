# Local-first

DumpSock's defining property: the bytes go from your phone to your disk and **nowhere else**. No cloud round-trip. No telemetry. No account.

## What "local-first" means in practice

- The iPhone is plugged into your Mac via USB.
- DumpSock talks to it over Apple's standard AFC service (the same one Finder uses for "Files on This iPhone").
- Each photo / video is read off the device in a single bulk stream.
- DumpSock writes it directly to the destination folder you chose.
- No intermediate cloud, no third-party server, no API endpoint.

There is no "DumpSock cloud" because there is no DumpSock cloud. There never will be.

## Why we built it this way

- **Privacy by architecture.** Things that don't exist can't be subpoenaed, leaked, or breached.
- **Speed.** USB 3 maxes out at ~5 Gbps — orders of magnitude faster than uploading 70 GB of video over a home internet connection.
- **Cost.** Cloud storage you pay for, even if your data never moves. Your SSD you buy once.
- **Trust.** Cloud accounts get suspended. Subscriptions lapse. Companies pivot. Disks you own keep working as long as the disk does.
- **Reproducibility.** With cloud, you have one copy at the mercy of someone else's uptime. Locally, you can pull twice into two destinations and have two physical copies.

## What this means for you

- DumpSock works without internet.
- DumpSock works behind any firewall.
- DumpSock can run on an air-gapped machine if you want.
- You can rebuild your photo library from the destination folder without ever opening DumpSock again — the on-disk layout is a vanilla folder tree of standard `.heic` / `.mov` / `.jpg` files.
