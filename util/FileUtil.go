package util

import (
	"bufio"
	"fmt"
	"github.com/beevik/etree"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetUserCurrent 获取当前用户的用户名
func GetUserCurrent() (string, error) {
	cmd := exec.Command("cmd", "/c", "echo", "%USERNAME%")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetDesktopPath 获取桌面路径
func GetDesktopPath(username string) (string, error) {
	desktopCmd := exec.Command("cmd", "/c", "echo", "%USERPROFILE%\\Desktop")
	desktopCmd.Env = append(os.Environ(), "USERNAME="+username)
	output, err := desktopCmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func ReadChannelIDs(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var channelIDs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		channelIDs = append(channelIDs, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return channelIDs, nil
}

// FileExists 检查文件是否存在
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// GetFileName 获取文件名
func GetFileName(filePath string) string {
	return filepath.Base(filePath)
}

func CopyFolderContents(source, destination string) error {
	// 遍历源文件夹
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 获取目标文件/文件夹路径
		destinationPath := filepath.Join(destination, path[len(source):])

		// 判断是文件夹还是文件
		if info.IsDir() {
			// 创建目标文件夹
			err := os.MkdirAll(destinationPath, info.Mode())
			if err != nil {
				return err
			}
		} else {
			// 复制文件
			err := CopyFile(path, destinationPath)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func CopyFile(source, destination string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	return nil
}

func GetFilesInDir(dirPath string) ([]string, error) {
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func FilterApkFiles(files []string) []string {
	var apkFiles []string
	for _, file := range files {
		if strings.HasSuffix(file, ".apk") {
			apkFiles = append(apkFiles, file)
		}
	}
	return apkFiles
}

func FilterTxtFiles(files []string) []string {
	var apkFiles []string
	for _, file := range files {
		if strings.HasSuffix(file, ".txt") {
			apkFiles = append(apkFiles, file)
		}
	}
	return apkFiles
}
func FilterJksFiles(files []string) []string {
	var apkFiles []string
	for _, file := range files {
		if strings.HasSuffix(file, ".jks") {
			apkFiles = append(apkFiles, file)
		}
	}
	return apkFiles
}
func UpdateXml(manifestPath, key, value string) error {
	//manifestPath := ""

	xmlData, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		fmt.Println("读取文件错误:", err)
		return err
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromString(string(xmlData)); err != nil {
		fmt.Println("解析XML错误:", err)
		return err
	}

	// 查找 android:name="UMENG_CHANNEL" 的 <meta-data> 标签
	xmlPath := fmt.Sprintf("//meta-data[@android:name='%s']", key)
	metaDataElem := doc.FindElement(xmlPath)
	if metaDataElem == nil {
		fmt.Println("错误: 未找到 <meta-data> 标签.")
		return err
	}

	// 将 android:value 属性值修改为 "baidu"
	metaDataElem.CreateAttr("android:value", value)

	// 将修改后的 XML 数据保存回文件
	updatedXMLData, _ := doc.WriteToString()
	if err := ioutil.WriteFile(manifestPath, []byte(updatedXMLData), 0644); err != nil {
		fmt.Println("写入文件错误:", err)
		return err
	}

	fmt.Println(value+": AndroidManifest.xml 文件已成功更新.")
	return nil
}

func GetAliasName(jksPath string) string {
	if strings.Contains(jksPath, "bjx_talents") {
		return "bjx.com.cn"
	} else {
		return "北极星电力头条签名文件"
	}
}

func GetKeyPassword(jksPath string) string {
	if strings.Contains(jksPath, "bjx_talents") {
		return "bjx.com.cn"
	} else {
		return "bjx123"
	}
}
func GetStorePassword(jksPath string) string {
	if strings.Contains(jksPath, "bjx_talents") {
		return "bjx.com.cn"
	} else {
		return "bjx123"
	}
}
