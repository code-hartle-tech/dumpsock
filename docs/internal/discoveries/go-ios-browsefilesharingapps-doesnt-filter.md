# go-ios `BrowseFileSharingApps()` doesn't actually filter

## Conclusion

Despite the name, `installationproxy.BrowseFileSharingApps()` in go-ios v1.0.213 returns **every** installed app (user + system), not just apps with `UIFileSharingEnabled=YES`. Source dive proves it sends `{Command: "Browse"}` with no app-type filter and no device-side `UIFileSharingEnabled` filter. The name is aspirational, not actual.

Symptom: the GUI's Browse → App data picker showed `com.apple.Fitness`, `com.apple.icq`, and the rest of the Apple system bundles. Every attempt to vend one of those failed with `InstallationLookupFailed` or `EOF`.

## What we tested

1. Called `conn.BrowseFileSharingApps()` against a real device → got back 200+ apps including system bundles.
2. Inspected the implementation in the module cache (see source proof below).
3. Switched to `conn.BrowseUserApps()` (which DOES filter on `ApplicationType=User` device-side) + a host-side `ai.UIFileSharingEnabled()` filter → got back ~15 third-party apps, all reachable via house_arrest.

## Source proof

```
/Users/vz/go/pkg/mod/github.com/danielpaulus/go-ios@v1.0.213/ios/installationproxy/installationproxy.go:65
    func (conn *Connection) BrowseFileSharingApps() ([]AppInfo, error) {
        return conn.browseApps(browseApps("Filesharing", true))
    }

installationproxy.go:161
    func browseApps(applicationType string, showLaunchProhibitedApps bool) map[string]interface{} {
        clientOptions := map[string]any{}
        if applicationType != "" && applicationType != "Filesharing" {  // ← skipped here!
            clientOptions["ApplicationType"] = applicationType
        }
        if showLaunchProhibitedApps {
            clientOptions["ShowLaunchProhibitedApps"] = true
        }
        return map[string]interface{}{"ClientOptions": clientOptions, "Command": "Browse"}
    }
```

When `applicationType == "Filesharing"`, the function explicitly SKIPS setting `ApplicationType` in `clientOptions`. Result: `{Command: "Browse", ClientOptions: {ShowLaunchProhibitedApps: true}}` — no app-type filter, no UIFileSharingEnabled filter, every app comes back.

## Canonical fix

Don't trust `BrowseFileSharingApps`. Use `BrowseUserApps` (drops system bundles device-side) and apply `ai.UIFileSharingEnabled()` host-side:

```go
apps, err := conn.BrowseUserApps()
if err != nil { ... }
for _, ai := range apps {
    if !ai.UIFileSharingEnabled() {
        continue
    }
    // …
}
```

## Where in the code

- `internal/gui/app.go::ListFileSharingApps` (the only caller; now does the two-step filter).

## Upstream

Could file a PR retitling the function to `BrowseFileSharingAppsButDoesNotActuallyFilter`, or simply documenting the current behavior. Have not filed. Better fix: have `BrowseFileSharingApps` do `BrowseUserApps` + UIFileSharingEnabled filter under the hood and match the name.

## Related

- [`house_arrest.New` hardcodes VendContainer](./go-ios-house-arrest-hardcodes-vendcontainer) — the other misnamed surface from the same upstream package set.
