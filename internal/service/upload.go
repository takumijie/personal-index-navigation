package service

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type UploadService struct {
	root          string
	maxImageBytes int64
	maxMusicBytes int64
}

func NewUploadService(root string, maxImageBytes, maxMusicBytes int64) *UploadService {
	return &UploadService{root: root, maxImageBytes: maxImageBytes, maxMusicBytes: maxMusicBytes}
}

func (s *UploadService) Save(kind string, header *multipart.FileHeader) (string, error) {
	if kind != "images" && kind != "music" && kind != "lyrics" {
		return "", invalid("上传类型无效")
	}
	limit := s.maxImageBytes
	if kind == "music" {
		limit = s.maxMusicBytes
	} else if kind == "lyrics" {
		limit = 1 << 20
	}
	if header.Size <= 0 || header.Size > limit {
		return "", invalid("上传文件为空或超过大小限制")
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExtension(kind, ext) {
		return "", invalid("上传文件扩展名不受支持")
	}
	file, err := header.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()
	if kind == "lyrics" {
		data, err := io.ReadAll(io.LimitReader(file, limit+1))
		if err != nil || len(data) == 0 || int64(len(data)) > limit || !utf8.Valid(data) || !plainText(string(data), true) {
			return "", invalid("歌词必须是有效的 UTF-8 文本")
		}
	} else {
		prefix := make([]byte, 512)
		n, err := io.ReadFull(file, prefix)
		if err != nil && err != io.ErrUnexpectedEOF {
			return "", invalid("无法读取上传文件")
		}
		prefix = prefix[:n]
		if !validFileContent(kind, ext, prefix, http.DetectContentType(prefix)) {
			return "", invalid("上传文件 MIME 或内容无效")
		}
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	dir := filepath.Join(s.root, kind)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	var output *os.File
	var name string
	for range 3 {
		random := make([]byte, 16)
		if _, err := rand.Read(random); err != nil {
			return "", err
		}
		name = hex.EncodeToString(random) + ext
		output, err = os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			break
		}
		if !os.IsExist(err) {
			return "", err
		}
	}
	if output == nil {
		return "", fmt.Errorf("无法分配上传文件名")
	}
	written, copyErr := io.Copy(output, io.LimitReader(file, limit+1))
	closeErr := output.Close()
	if copyErr != nil || closeErr != nil || written > limit {
		_ = os.Remove(filepath.Join(dir, name))
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		return "", invalid("上传文件超过大小限制")
	}
	return "/storage/" + kind + "/" + name, nil
}

func (s *UploadService) DeleteMusic(rawURL string) error {
	const prefix = "/storage/music/"
	if !strings.HasPrefix(rawURL, prefix) {
		return invalid("本地音频路径无效")
	}
	name := strings.TrimPrefix(rawURL, prefix)
	ext := filepath.Ext(name)
	encoded := strings.TrimSuffix(name, ext)
	if len(encoded) != 32 || len(name) != 32+len(ext) || ext != strings.ToLower(ext) ||
		encoded != strings.ToLower(encoded) || !allowedExtension("music", ext) {
		return invalid("本地音频路径无效")
	}
	if _, err := hex.DecodeString(encoded); err != nil {
		return invalid("本地音频路径无效")
	}
	filename := filepath.Join(s.root, "music", name)
	info, err := os.Lstat(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return invalid("本地音频不是普通文件")
	}
	if err := os.Remove(filename); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func allowedExtension(kind, ext string) bool {
	if kind == "images" {
		return oneOf(ext, ".jpg", ".jpeg", ".png", ".gif", ".webp")
	}
	if kind == "lyrics" {
		return ext == ".lrc"
	}
	return oneOf(ext, ".mp3", ".flac", ".wav", ".ogg")
}

func validFileContent(kind, ext string, data []byte, mime string) bool {
	if kind == "images" {
		validMIME := oneOf(mime, "image/jpeg", "image/png", "image/gif", "image/webp")
		if !validMIME {
			return false
		}
		switch ext {
		case ".jpg", ".jpeg":
			return mime == "image/jpeg" && len(data) >= 3 && bytes.Equal(data[:3], []byte{0xff, 0xd8, 0xff})
		case ".png":
			return mime == "image/png" && len(data) >= 8 && bytes.Equal(data[:8], []byte("\x89PNG\r\n\x1a\n"))
		case ".gif":
			return mime == "image/gif" && len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a")
		case ".webp":
			return mime == "image/webp" && len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP"
		}
	}
	if !oneOf(mime, "audio/mpeg", "audio/flac", "audio/x-flac", "audio/wav", "audio/wave", "audio/x-wav", "audio/ogg", "application/ogg", "application/octet-stream") {
		return false
	}
	switch ext {
	case ".mp3":
		return len(data) >= 3 && (string(data[:3]) == "ID3" || (data[0] == 0xff && data[1]&0xe0 == 0xe0))
	case ".flac":
		return len(data) >= 4 && string(data[:4]) == "fLaC"
	case ".wav":
		return len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE"
	case ".ogg":
		return len(data) >= 4 && string(data[:4]) == "OggS"
	}
	return false
}
