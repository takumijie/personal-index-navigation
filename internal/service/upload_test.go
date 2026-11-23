package service

import (
	"net/http"
	"testing"
)

func TestImageUploadSupportsICO(t *testing.T) {
	if !allowedExtension("images", ".ico") {
		t.Fatal("图片上传应支持 .ico 扩展名")
	}
	icon := []byte{0x00, 0x00, 0x01, 0x00, 0x01, 0x00}
	mime := http.DetectContentType(icon)
	if !validFileContent("images", ".ico", icon, mime) {
		t.Fatalf("有效 ICO 文件应通过内容和 MIME 校验，检测 MIME 为 %s", mime)
	}

	invalidCases := []struct {
		name string
		data []byte
		mime string
	}{
		{name: "空图标目录", data: []byte{0x00, 0x00, 0x01, 0x00, 0x00, 0x00}, mime: "image/x-icon"},
		{name: "错误文件头", data: []byte{0x00, 0x00, 0x02, 0x00, 0x01, 0x00}, mime: "image/x-icon"},
		{name: "错误 MIME", data: icon, mime: "image/png"},
	}
	for _, testCase := range invalidCases {
		t.Run(testCase.name, func(t *testing.T) {
			if validFileContent("images", ".ico", testCase.data, testCase.mime) {
				t.Fatal("无效 ICO 文件应被拒绝")
			}
		})
	}
}
