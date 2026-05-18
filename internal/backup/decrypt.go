package backup

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/pbkdf2"
)

// DecryptFile reverses EncryptFile. Reads src (must be DSAES2 or
// DSAES1 format) and writes the recovered plaintext to dst. Caller
// chooses dst; if it ends in .zip we end up with a vanilla zip the
// user can drag into Finder. Returns dst on success.
//
// Streaming: each chunk is decrypted independently with its own nonce
// + auth tag, so memory usage is bounded at chunkSize (1 MiB for
// DSAES2). DSAES1 is single-shot; we read the whole file for that
// path since the format requires it.
//
// onProgress (nullable) is invoked periodically with PackageProgress
// carrying Phase="decrypting", BytesDone=plaintext-bytes-written,
// BytesTotal=size of src (a slight over-estimate since each chunk
// includes a 12-byte nonce + 16-byte tag on top of plaintext — close
// enough for a progress bar).
func DecryptFile(src, dst, password string, onProgress func(PackageProgress)) (string, error) {
	if password == "" {
		return "", errors.New("DecryptFile: empty password")
	}
	in, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("open src: %w", err)
	}
	defer in.Close()

	// Read the 7-byte magic to pick the format.
	magic := make([]byte, 7)
	if _, err := io.ReadFull(in, magic); err != nil {
		return "", fmt.Errorf("read magic: %w", err)
	}
	// Stat src so the progress denominator is real.
	srcSize := int64(0)
	if fi, _ := in.Stat(); fi != nil {
		srcSize = fi.Size()
	}
	if onProgress != nil {
		onProgress(PackageProgress{Phase: "decrypting", BytesTotal: srcSize})
	}

	switch string(magic) {
	case dsaes2Magic:
		return decryptDSAES2(in, dst, password, srcSize, onProgress)
	case "DSAES1\n":
		return decryptDSAES1(in, dst, password, srcSize, onProgress)
	default:
		return "", fmt.Errorf("not a DumpSock encrypted archive (magic = %q)", magic)
	}
}

// decryptDSAES2 is the streaming, chunked format. Layout (after the
// 7-byte magic already read):
//
//	salt(16) | iters(uint32 BE) | chunkSize(uint32 BE) | [chunks...]
//
// each chunk = nonce(12) || ciphertext+tag(≤chunkSize+16).
func decryptDSAES2(in *os.File, dst, password string, srcSize int64, onProgress func(PackageProgress)) (string, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(in, salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}
	hdr := make([]byte, 8)
	if _, err := io.ReadFull(in, hdr); err != nil {
		return "", fmt.Errorf("read iters/chunkSize: %w", err)
	}
	iters := int(binary.BigEndian.Uint32(hdr[0:4]))
	chunkSize := int(binary.BigEndian.Uint32(hdr[4:8]))
	if iters <= 0 || chunkSize <= 0 || chunkSize > 64<<20 {
		return "", fmt.Errorf("implausible DSAES2 header (iters=%d, chunkSize=%d)", iters, chunkSize)
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
	nonceSize := aead.NonceSize()
	tagSize := aead.Overhead() // 16 for GCM

	out, err := os.Create(dst)
	if err != nil {
		return "", fmt.Errorf("create dst: %w", err)
	}
	defer out.Close()

	// Read until EOF. Each iteration: nonce + up-to-chunkSize+tag.
	maxCT := chunkSize + tagSize
	buf := make([]byte, maxCT)
	nonce := make([]byte, nonceSize)
	var bytesDone int64
	for chunkNum := 0; ; chunkNum++ {
		_, err := io.ReadFull(in, nonce)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read nonce chunk %d: %w", chunkNum, err)
		}
		// Read up to maxCT bytes of ciphertext+tag.
		n, err := io.ReadFull(in, buf)
		if err == io.ErrUnexpectedEOF {
			// Final short chunk — that's fine; n holds what we got.
		} else if err != nil && err != io.EOF {
			return "", fmt.Errorf("read ct chunk %d: %w", chunkNum, err)
		}
		if n < tagSize {
			return "", fmt.Errorf("chunk %d too short to contain a GCM tag", chunkNum)
		}
		plain, err := aead.Open(nil, nonce, buf[:n], nil)
		if err != nil {
			return "", fmt.Errorf("decrypt chunk %d: %w (wrong password or corrupt archive)", chunkNum, err)
		}
		if _, err := out.Write(plain); err != nil {
			return "", fmt.Errorf("write chunk %d: %w", chunkNum, err)
		}
		bytesDone += int64(len(plain))
		if onProgress != nil {
			onProgress(PackageProgress{
				Phase:      "decrypting",
				BytesDone:  bytesDone,
				BytesTotal: srcSize,
			})
		}
	}
	return dst, nil
}

// decryptDSAES1 is the single-shot legacy format (≤ 2026-05-18). Kept
// so users with existing .zip.aes from old builds can still recover.
// Layout (after the 7-byte magic):
//
//	salt(16) | iters(uint32 BE) | nonce(12) | ct‖tag
func decryptDSAES1(in *os.File, dst, password string, srcSize int64, onProgress func(PackageProgress)) (string, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(in, salt); err != nil {
		return "", err
	}
	itersBuf := make([]byte, 4)
	if _, err := io.ReadFull(in, itersBuf); err != nil {
		return "", err
	}
	iters := int(binary.BigEndian.Uint32(itersBuf))
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(in, nonce); err != nil {
		return "", err
	}
	ct, err := io.ReadAll(in)
	if err != nil {
		return "", err
	}

	key := pbkdf2.Key([]byte(password), salt, iters, 32, sha256.New)
	block, _ := aes.NewCipher(key)
	aead, _ := cipher.NewGCM(block)
	plain, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt DSAES1: %w (wrong password or corrupt archive)", err)
	}
	if err := os.WriteFile(dst, plain, 0o644); err != nil {
		return "", err
	}
	// DSAES1 is single-shot — fire one final progress at 100% so the
	// bar lights up green instead of stalling at the magic-read tick.
	if onProgress != nil {
		onProgress(PackageProgress{
			Phase:      "decrypting",
			BytesDone:  srcSize,
			BytesTotal: srcSize,
		})
	}
	return dst, nil
}
