package util

import (
	"fmt"
	"github.com/beevik/etree"
	"io/ioutil"
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
	fmt.Printf("正在备份[%s]...\n", GetFileName(newFolderPath))
	err1 := CopyFolderContents(tempDir, newFolderPath)
	if err1 != nil {
		return "备份反编译结果文件失败"
	}
	if err := UpdateXml(newFolderPath+"\\AndroidManifest.xml", channelKey, channelID); err != nil {
		return fmt.Sprintf("修改AndroidManifest.xml时出错: %v\n", err)
	}
	fmt.Printf("已完成[%s]渠道信息修改！\n", channelID)
	newApkName := fmt.Sprintf("%s-%s", apkName, channelID)
	outputAPKPath := fmt.Sprintf("%s\\%s.apk", outputDir, newApkName)
	fmt.Printf("正在重新打包 [%s] ...\n", GetFileName(outputAPKPath))
	cmd := exec.Command(apkToolPath, "b", newFolderPath, "-o", outputAPKPath)
	_, runErr := cmd.CombinedOutput()
	if runErr != nil {
		fmt.Printf("APK重新打包过程中出错: %v\n", runErr)
		return fmt.Sprintf("APK重新打包过程中出错: %v\n", runErr)
	}
	fmt.Printf("正在删除临时文件:%s...\n", GetFileName(newFolderPath))
	err := os.RemoveAll(newFolderPath)
	if err != nil {
		fmt.Printf("删除临时文件时出错: %v\n", err)
		return "删除临时文件时出错"
	}
	return fmt.Sprintf("[%s]渠道打包完成...\n", channelID)
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
