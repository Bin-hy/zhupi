package sdkassets

import "embed"

//go:embed all:dist
var files embed.FS

// Read 返回构建产物；未执行 SDK 构建时只会存在 .gitkeep，读取会失败。
func Read(name string) ([]byte, error) {
	return files.ReadFile("dist/" + name)
}
