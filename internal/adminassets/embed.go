package adminassets

import "embed"

//go:embed all:dist
var files embed.FS

// Read 返回管理后台构建产物；未构建时仅存在 .gitkeep。
func Read(name string) ([]byte, error) {
	return files.ReadFile("dist/" + name)
}
