//go:build darwin

// Package macauth wraps macOS LocalAuthentication + Keychain so DumpSock
// can stash the user's backup password behind a Touch ID prompt. The
// stored item gets `kSecAttrAccessControl = BiometryCurrentSet`, which
// means:
//
//   - retrieval pops the system Touch ID sheet (or login-password
//     fallback if biometry is unavailable/locked-out)
//   - adding a new fingerprint on the Mac invalidates the item (the
//     ACL is bound to the current biometric set)
//   - the bytes live in the Secure Enclave on T2 / Apple Silicon Macs
//
// The actual encryption-key derivation in package.go is unchanged —
// the password we hide here is still the user's typed string. Touch
// ID is a convenience layer (Apple's autofill pattern), not a new
// cryptographic primitive. Archives stay portable: a user can copy
// the .zip.aes to another Mac and type the same password to decrypt.
//
// CGo bridge synthesized from the three biometric-research agents
// fired 2026-05-18 (see project memory feedback_nonnegotiables.md).
package macauth

/*
#cgo CFLAGS: -x objective-c -fmodules -fblocks -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework LocalAuthentication -framework Security -framework CoreFoundation

#include <Security/Security.h>
#include <LocalAuthentication/LocalAuthentication.h>

// canUseBiometrics returns 1 if Touch ID / Face ID is available on this
// device AND enrolled. We use LAPolicyDeviceOwnerAuthentication so a
// Mac without a Touch ID sensor (but with a login password) still
// returns 1 — the user just gets the password prompt instead.
int dsCanUseBiometrics(void) {
    LAContext *ctx = [[LAContext alloc] init];
    NSError *err = nil;
    BOOL ok = [ctx canEvaluatePolicy:LAPolicyDeviceOwnerAuthentication error:&err];
    return ok ? 1 : 0;
}

// dsStorePassword writes `password` into the keychain under
// (service=tech.hartle.dumpsock, account=backup-password-v1), gated by
// biometry-current-set. Overwrites any existing item silently.
OSStatus dsStorePassword(const char *password, size_t n) {
    CFErrorRef e = NULL;
    SecAccessControlRef ac = SecAccessControlCreateWithFlags(
        kCFAllocatorDefault,
        kSecAttrAccessibleWhenPasscodeSetThisDeviceOnly,
        kSecAccessControlBiometryCurrentSet, &e);
    if (!ac) return -1;

    CFStringRef svc  = CFStringCreateWithCString(NULL, "tech.hartle.dumpsock",   kCFStringEncodingUTF8);
    CFStringRef acct = CFStringCreateWithCString(NULL, "backup-password-v1",     kCFStringEncodingUTF8);
    CFDataRef   data = CFDataCreate(NULL, (const UInt8*)password, n);

    const void *k[] = {kSecClass, kSecAttrService, kSecAttrAccount, kSecValueData, kSecAttrAccessControl};
    const void *v[] = {kSecClassGenericPassword, svc, acct, data, ac};
    CFDictionaryRef q = CFDictionaryCreate(NULL, k, v, 5, NULL, NULL);

    // Idempotent overwrite — delete-then-add is the simplest stable path.
    const void *dk[] = {kSecClass, kSecAttrService, kSecAttrAccount};
    const void *dv[] = {kSecClassGenericPassword, svc, acct};
    CFDictionaryRef dq = CFDictionaryCreate(NULL, dk, dv, 3, NULL, NULL);
    SecItemDelete(dq);
    CFRelease(dq);

    OSStatus st = SecItemAdd(q, NULL);
    CFRelease(q); CFRelease(data); CFRelease(acct); CFRelease(svc); CFRelease(ac);
    return st;
}

// dsLoadPassword presents the Touch ID sheet (or login-password
// fallback). On success, copies the stored password bytes into `out`
// and sets `outlen`. Caller must free `out`.
OSStatus dsLoadPassword(const char *reason, void **out, size_t *outlen) {
    LAContext *ctx = [[LAContext alloc] init];
    if (reason && *reason) {
        ctx.localizedReason = [NSString stringWithUTF8String:reason];
    }

    CFStringRef svc  = CFStringCreateWithCString(NULL, "tech.hartle.dumpsock", kCFStringEncodingUTF8);
    CFStringRef acct = CFStringCreateWithCString(NULL, "backup-password-v1",   kCFStringEncodingUTF8);

    const void *k[] = {kSecClass, kSecAttrService, kSecAttrAccount,
                       kSecReturnData, kSecUseAuthenticationContext};
    const void *v[] = {kSecClassGenericPassword, svc, acct, kCFBooleanTrue, (__bridge CFTypeRef)ctx};
    CFDictionaryRef q = CFDictionaryCreate(NULL, k, v, 5, NULL, NULL);

    CFTypeRef result = NULL;
    OSStatus st = SecItemCopyMatching(q, &result);  // blocks on Touch ID sheet
    CFRelease(q); CFRelease(acct); CFRelease(svc);
    if (st == errSecSuccess && result) {
        CFDataRef d = (CFDataRef)result;
        *outlen = CFDataGetLength(d);
        *out = malloc(*outlen);
        memcpy(*out, CFDataGetBytePtr(d), *outlen);
        CFRelease(d);
    }
    return st;
}

// dsHasPassword checks whether an item exists (without prompting).
int dsHasPassword(void) {
    CFStringRef svc  = CFStringCreateWithCString(NULL, "tech.hartle.dumpsock", kCFStringEncodingUTF8);
    CFStringRef acct = CFStringCreateWithCString(NULL, "backup-password-v1",   kCFStringEncodingUTF8);

    const void *k[] = {kSecClass, kSecAttrService, kSecAttrAccount,
                       kSecMatchLimit, kSecUseAuthenticationUI};
    const void *v[] = {kSecClassGenericPassword, svc, acct,
                       kSecMatchLimitOne, kSecUseAuthenticationUIFail};
    CFDictionaryRef q = CFDictionaryCreate(NULL, k, v, 5, NULL, NULL);

    OSStatus st = SecItemCopyMatching(q, NULL);
    CFRelease(q); CFRelease(acct); CFRelease(svc);

    // errSecInteractionNotAllowed (-25308) means "the item exists, we
    // just didn't unlock it" — that's still a "yes, item exists" for
    // our purposes (UI gating "Touch ID enrolled" indicator).
    return (st == errSecSuccess || st == -25308) ? 1 : 0;
}

// dsClearPassword removes the keychain item. No prompt.
OSStatus dsClearPassword(void) {
    CFStringRef svc  = CFStringCreateWithCString(NULL, "tech.hartle.dumpsock", kCFStringEncodingUTF8);
    CFStringRef acct = CFStringCreateWithCString(NULL, "backup-password-v1",   kCFStringEncodingUTF8);

    const void *k[] = {kSecClass, kSecAttrService, kSecAttrAccount};
    const void *v[] = {kSecClassGenericPassword, svc, acct};
    CFDictionaryRef q = CFDictionaryCreate(NULL, k, v, 3, NULL, NULL);

    OSStatus st = SecItemDelete(q);
    CFRelease(q); CFRelease(acct); CFRelease(svc);
    return st;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

// Available reports whether the OS can present a biometric (or
// password-fallback) authentication sheet at all. False on Linux/
// Windows builds, and on Macs without a login password set.
func Available() bool {
	return C.dsCanUseBiometrics() == 1
}

// StorePassword stashes `password` in Keychain behind a biometry-gated
// access control. Overwrites any prior value.
func StorePassword(password string) error {
	if password == "" {
		return errors.New("StorePassword: empty password")
	}
	b := []byte(password)
	st := int32(C.dsStorePassword((*C.char)(unsafe.Pointer(&b[0])), C.size_t(len(b))))
	if st != 0 {
		return fmt.Errorf("keychain store failed: %s", osStatusName(st))
	}
	return nil
}

// osStatusName translates the OSStatus codes we hit most often into a
// human-readable explanation. Surfaces what's actually wrong instead
// of leaving the operator staring at "keychain store failed."
func osStatusName(st int32) string {
	switch st {
	case -34018:
		return "errSecMissingEntitlement (-34018) — DumpSock isn't signed with a Developer ID. Touch ID-gated Keychain entries require the app to be signed; this works once Phase 4 codesigning ships. Use the typed password flow for now."
	case -25291:
		return "errSecNotAvailable (-25291) — Keychain isn't available (login keychain locked or missing). Unlock it in Keychain Access and retry."
	case -25299:
		return "errSecDuplicateItem (-25299) — A prior Touch ID entry exists and couldn't be replaced. This shouldn't happen since we delete-before-add; report if it persists."
	case -25300:
		return "errSecItemNotFound (-25300) — Expected when no item exists yet; should never appear on store."
	case -25303:
		return "errSecNoSuchKeychain (-25303) — Default keychain not found."
	case -25308:
		return "errSecInteractionNotAllowed (-25308) — Keychain wouldn't unlock without UI; should not happen on store."
	case -1:
		return "-1 — SecAccessControlCreateWithFlags returned NULL. Usually means the binary isn't signed (BiometryCurrentSet ACLs require Developer ID signing on macOS 11+)."
	default:
		return fmt.Sprintf("OSStatus %d (look up in <Security/SecBase.h>)", st)
	}
}

// LoadPassword presents the Touch ID sheet (or login-password fallback)
// and returns the stored password. The system blocks the calling
// goroutine while the sheet is up — Wails-bound methods are off the
// main UI thread, so this is safe to call directly from a binding.
func LoadPassword(reason string) (string, error) {
	if reason == "" {
		reason = "Unlock your DumpSock backup password"
	}
	cReason := C.CString(reason)
	defer C.free(unsafe.Pointer(cReason))
	var out unsafe.Pointer
	var n C.size_t
	st := C.dsLoadPassword(cReason, &out, &n)
	if st != 0 || out == nil {
		return "", errors.New("keychain load denied or missing")
	}
	defer C.free(out)
	return C.GoStringN((*C.char)(out), C.int(n)), nil
}

// HasPassword returns whether a password is currently enrolled. Does
// NOT prompt the user — safe to call on every app launch for UI gating.
func HasPassword() bool {
	return C.dsHasPassword() == 1
}

// ClearPassword removes the enrolled password from Keychain.
func ClearPassword() error {
	st := C.dsClearPassword()
	if st != 0 && st != -25300 /* errSecItemNotFound */ {
		return errors.New("keychain clear failed")
	}
	return nil
}
