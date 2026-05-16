//go:build ignore

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

const windowsReadme = `Internal Tool

QQ缓存导入:
internal-tool.exe qq-cache-import -dir test_upload

强制覆盖已存在账号缓存字段:
internal-tool.exe qq-cache-import -dir test_upload -force

说明:
- zip文件名格式: {qq uin}----{pwd}.zip
- 默认不覆盖已存在账号，已存在会跳过，不上传zip
- 把待导入zip放到 test_upload 目录后执行命令
`

func main() {
	distDir := filepath.Join(".", "dist")
	pkgDir := filepath.Join(distDir, "internal-tool-windows")
	exePath := filepath.Join(pkgDir, "internal-tool.exe")
	zipPath := filepath.Join(distDir, "internal-tool-windows.zip")

	if err := os.RemoveAll(pkgDir); err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(pkgDir, "test_upload"), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "README.txt"), []byte(windowsReadme), 0o644); err != nil {
		fatal(err)
	}

	cmd := exec.Command("go", "build", "-o", exePath, ".")
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal(err)
	}

	if err := os.Remove(zipPath); err != nil && !os.IsNotExist(err) {
		fatal(err)
	}
	if err := zipDir(pkgDir, zipPath); err != nil {
		fatal(err)
	}
	fmt.Println("Windows包已生成:", zipPath)
}

func zipDir(srcDir string, zipPath string) error {
	out, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	return filepath.WalkDir(srcDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(filepath.Dir(srcDir), path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if rel == filepath.Base(srcDir) {
				return nil
			}
			header := &zip.FileHeader{
				Name:   rel + "/",
				Method: zip.Store,
			}
			_, err := zw.CreateHeader(header)
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = rel
		header.Method = zip.Deflate
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(writer, in)
		return err
	})
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
