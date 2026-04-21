package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func readFile(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file error: %v", err)
	}
	return data, nil
}

func readSqliteData(dbPath string) ([]byte, error) {
	fmt.Println("Reading from tk_file:", dbPath)
	// 使用 sqlite3 命令读取二进制数据
	cmd := exec.Command("sqlite3", dbPath, "SELECT quote(tk_file) FROM tk_file LIMIT 1;")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("sqlite3 command error: %v", err)
	}

	// 清理输出数据
	hexData := strings.TrimSpace(string(output))
	fmt.Printf("Raw SQLite output: %q\n", hexData)

	// 移除 X' 前缀和 ' 后缀
	hexData = strings.TrimPrefix(hexData, "X'")
	hexData = strings.TrimSuffix(hexData, "'")
	hexData = strings.TrimSpace(hexData)

	fmt.Printf("Cleaned hex data: %q\n", hexData)

	// 解码十六进制字符串
	var result []byte
	for i := 0; i < len(hexData); i += 2 {
		if i+1 >= len(hexData) {
			break
		}
		byteStr := hexData[i : i+2]
		var b byte
		_, err := fmt.Sscanf(byteStr, "%02x", &b)
		if err != nil {
			return nil, fmt.Errorf("hex decode error for byte %s: %v", byteStr, err)
		}
		result = append(result, b)
	}

	fmt.Printf("Decoded %d bytes\n", len(result))
	return result, nil
}

// go run . -tk=/Users/fupeng/Workspace/test/aaa/databases/tk_file -wlogin=/Users/fupeng/Workspace/test/aaa/files/wlogin_device.dat
// java -jar SerializationDumper.jar -r out.dat > decode.dat
func main() {
	tkFile := flag.String("tk", "", "path to tk file")
	wloginFile := flag.String("wlogin", "", "path to wlogin_device.dat file")
	flag.Parse()

	if *tkFile == "" || *wloginFile == "" {
		fmt.Println("Usage: decrypt_cache -tk <tk_file_path> -wlogin <wlogin_device.dat_path>")
		flag.PrintDefaults()
		return
	}

	// Read wlogin_device.dat for GUID
	guid, err := readFile(*wloginFile)
	if err != nil {
		fmt.Printf("Error reading wlogin file: %v\n", err)
		return
	}

	// Read tk file using sqlite3
	tkData, err := readSqliteData(*tkFile)
	if err != nil {
		fmt.Printf("Error reading tk file: %v\n", err)
		return
	}

	// Decrypt the data
	c := NewTeaCipher(guid)
	result := c.Decrypt(tkData)

	// Write decrypted data to output file
	err = os.WriteFile("out.dat", result, 0644)
	if err != nil {
		fmt.Printf("Error writing output file: %v\n", err)
		return
	}

	fmt.Println("Decryption completed successfully. Output written to out.dat")
}
