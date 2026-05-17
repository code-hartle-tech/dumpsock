package backup

import (
	"archive/zip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/pbkdf2"

	// hash for PBKDF2-SHA256 imports lazily via crypto/sha256.
	_ "crypto/sha256"
	"crypto/sha256"
)

// PackageZip walks outputRoot and writes a sibling .zip archive that
// contains every file under it (skipping the .dumpsock-* sentinels so
// the archive is a clean media tree). Returns the path of the produced
// .zip. Honors ctx cancellation between files.
//
// Naming: outputRoot=/x/y/MyPhone → /x/y/MyPhone.zip. If that path is
// already taken, we suffix with -1, -2, ... until we find a free name.
func PackageZip(ctx context.Context, outputRoot string) (string, error) {
	if outputRoot == "" {
		return "", errors.New("PackageZip: empty outputRoot")
	}
	cleaned := filepath.Clean(outputRoot)
	parent := filepath.Dir(cleaned)
	leaf := filepath.Base(cleaned)
	zipPath := freeName(filepath.Join(parent, leaf+".zip"))

	out, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("create zip: %w", err)
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	walkErr := filepath.WalkDir(cleaned, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		name := d.Name()
		// Skip DumpSock-internal sentinels / staging dirs.
		if strings.HasPrefix(name, ".dumpsock") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip the zip itself if it happens to land under outputRoot
		// (defensive — we already chose a sibling path).
		if !d.IsDir() && filepath.Clean(p) == zipPath {
			return nil
		}
		rel, _ := filepath.Rel(cleaned, p)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			_, err := zw.Create(rel + "/")
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = rel
		header.Method = zip.Deflate
		w, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, f)
		_ = f.Close()
		return copyErr
	})
	closeErr := zw.Close()
	if walkErr != nil {
		return "", walkErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return zipPath, nil
}

// EncryptFile reads src, encrypts the bytes with AES-256-GCM where the
// key is PBKDF2-SHA256(password, salt, 200000 iters, 32 bytes), and
// writes the result to a sibling file with ".aes" appended.
//
// Container format (binary):
//
//	magic   "DSAES1\n"             7 bytes
//	salt    16 random bytes
//	iters   uint32 BE (PBKDF2 iterations)
//	nonce   12 random bytes (AES-GCM)
//	ct||tag = AES-GCM ciphertext (16-byte auth tag at the tail)
//
// Returns the path of the produced .aes file.
func EncryptFile(src, password string) (string, error) {
	if password == "" {
		return "", errors.New("EncryptFile: empty password")
	}
	plaintext, err := os.ReadFile(src)
	if err != nil {
		return "", fmt.Errorf("read src: %w", err)
	}

	const iters = 200_000
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("salt: %w", err)
	}
	key := pbkdf2.Key([]byte(password), salt, iters, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	dst := src + ".aes"
	out, err := os.Create(dst)
	if err != nil {
		return "", fmt.Errorf("create dst: %w", err)
	}
	defer out.Close()

	if _, err := out.Write([]byte("DSAES1\n")); err != nil {
		return "", err
	}
	if _, err := out.Write(salt); err != nil {
		return "", err
	}
	itersBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(itersBuf, iters)
	if _, err := out.Write(itersBuf); err != nil {
		return "", err
	}
	if _, err := out.Write(nonce); err != nil {
		return "", err
	}
	if _, err := out.Write(ciphertext); err != nil {
		return "", err
	}
	return dst, nil
}

// freeName returns base if base doesn't exist, otherwise base with a
// -1, -2, ... suffix inserted before the extension to find an unused
// filename. Bounded at 1000 attempts (then it just appends -N+1 anyway).
func freeName(base string) string {
	if _, err := os.Stat(base); err != nil {
		return base
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for i := 1; i < 1000; i++ {
		cand := fmt.Sprintf("%s-%d%s", stem, i, ext)
		if _, err := os.Stat(cand); err != nil {
			return cand
		}
	}
	return fmt.Sprintf("%s-%d%s", stem, 999, ext)
}
