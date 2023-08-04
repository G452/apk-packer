package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func PackAPK(tempDir, outputDir, apkName, channelIDs, apkToolPath string) string {
	parts := strings.Split(channelIDs, ",")
	channelKey := parts[0]
	channelID := parts[1]
	newFolderPath := filepath.Join(outputDir, "tempApk-"+channelID)
	if _, err := os.Stat(newFolderPath); err == nil {
		if err := os.RemoveAll(newFolderPath); err != nil {
			fmt.Printf("删除文件失败: %s\n", err)
			return "删除ApkTemp文件失败"
		}
		fmt.Printf("删除原文件夹tempApk-" + channelID + "\n")
	}
	errCopy := os.Mkdir(newFolderPath, 0755)
	if errCopy != nil {
		return "新建文件夹失败"
	}
	err1 := CopyFolderContents(tempDir, newFolderPath)
	if err1 != nil {
		return "复制反编译结果文件失败"
	}
	fmt.Printf("开始处理 [%s] 渠道\n", channelID)
	if err := UpdateXml(newFolderPath+"\\AndroidManifest.xml", channelKey, channelID); err != nil {
		return fmt.Sprintf("修改AndroidManifest.xml时出错: %v\n", err)
	}
	newApkName := fmt.Sprintf("%s-%s", apkName, channelID)
	outputAPKPath := fmt.Sprintf("%s\\%s.apk", outputDir, newApkName)
	cmd := exec.Command(apkToolPath, "b", newFolderPath, "-o", outputAPKPath)
	_, runErr := cmd.CombinedOutput()
	if runErr != nil {
		fmt.Printf("APK重新打包过程中出错: %v\n", runErr)
		return fmt.Sprintf("APK重新打包过程中出错: %v\n", runErr)
	}
	err := os.RemoveAll(newFolderPath)
	if err != nil {
		fmt.Printf("删除临时文件时出错: %v\n", err)
		return "删除临时文件时出错"
	}
	return fmt.Sprintf("[%s]%s渠道打包完成...\n", channelID)
}

func SignAPKsWithJks(apkFiles []string, jksPath string) {
	for _, apkFile := range apkFiles {
		keyAlias := GetAliasName(jksPath)
		keyPassword := GetKeyPassword(jksPath)
		storePassword := GetStorePassword(jksPath)
		cmd := exec.Command("jarsigner", "-verbose", "-sigalg", "SHA1withRSA", "-digestalg", "SHA1", "-keystore", jksPath, "-storepass", storePassword, "-keypass", keyPassword, apkFile, keyAlias)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("签名失败：%s: %v\nOutput:\n%s\n", apkFile, err, string(output))
		} else {
			fmt.Printf("APK %s 签名成功.\n", GetFileName(apkFile))
		}
	}
}
