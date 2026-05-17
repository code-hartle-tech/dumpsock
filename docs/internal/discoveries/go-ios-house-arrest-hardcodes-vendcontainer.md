# go-ios `house_arrest.New` hardcodes `VendContainer`

## Conclusion

`github.com/danielpaulus/go-ios/ios/house_arrest@v1.0.213`'s only constructor sends the `VendContainer` plist message. Apple gates `VendContainer` on debuggable / dev-signed apps; production-signed App-Store apps with `UIFileSharingEnabled=YES` ONLY respond to `VendDocuments`. The symptom is `InstallationLookupFailed` on every real-world app the user tries to browse.

We ship an in-repo wrapper at `internal/afc/housearrest.go::OpenAppContainer` that tries `VendDocuments` first, falls back to `VendContainer`. Until an upstream PR lands, this is the only path that works for App-Store apps.

## What we tested

1. Picked `com.dji.golite` (DJI GO Lite — App-Store-installed drone app, ships `UIFileSharingEnabled=YES`).
2. Routed through `gohouse.New(dev, "com.dji.golite")` → instant `InstallationLookupFailed`.
3. Wrote a quick patch that sends `{Command: "VendDocuments", Identifier: "com.dji.golite"}` on the same lockdown service → `{Status: "Complete"}` → AFC channel onto the app's Documents folder.

Same pattern reproduced against several other UIFileSharingEnabled apps. VendContainer fails uniformly; VendDocuments succeeds uniformly.

## Source proof

Local module cache:

```
/Users/vz/go/pkg/mod/github.com/danielpaulus/go-ios@v1.0.213/ios/house_arrest/house_arrest.go:30
    vendContainer := map[string]interface{}{"Command": "VendContainer", "Identifier": bundleID}
```

No `VendDocuments` constant anywhere in the file. No second constructor either.

## Canonical fix

`internal/afc/housearrest.go` reimplements the protocol surface using go-ios's lower-level lockdown + AFC primitives:

```go
func openAppVend(dev ios.DeviceEntry, bundleID, command string) (*afc.Client, error) {
    conn, err := ios.ConnectToService(dev, "com.apple.mobile.house_arrest")
    ...
    req := map[string]interface{}{"Command": command, "Identifier": bundleID}
    msg, _ := codec.Encode(req)
    conn.Send(msg)
    response, _ := codec.Decode(conn.Reader())
    // unmarshal, check Status == "Complete", return afc.NewFromConn(conn)
}
```

`OpenAppContainer(udid, bundleID)` tries `command="VendDocuments"` first, falls back to `"VendContainer"`.

## Where in the code

- `internal/afc/housearrest.go` (the wrapper)
- `internal/gui/app.go::BrowseApp` (the only caller; routes through the wrapper instead of `gohouse.New`)

## Upstream

Should be a ~15-LOC PR to `github.com/danielpaulus/go-ios/ios/house_arrest` adding a `New2(device, bundleID, command)` constructor or a `VendDocuments(device, bundleID)` helper. Not filed yet. Tracked locally as task #21.

## Related

- [go-ios `BrowseFileSharingApps` doesn't filter](./go-ios-browsefilesharingapps-doesnt-filter) — the *other* misnamed go-ios function we worked around in the same session.
- [iOS 15+ hardening cliffs](./ios-15-hardening-cliffs) — the broader iOS permission landscape this fits into.
