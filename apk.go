package main

import (
	"apk-packer/util"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func main() {
	startTime := time.Now() // 记录开始时间
	currentUser, err := util.GetUserCurrent()
	currentDesk, err := util.GetDesktopPath(currentUser)
	if err != nil {
		fmt.Printf("未找到用户桌面: %v\n", err)
		return
	}
	apkFilePath := currentDesk + "\\funckapk\\baseApk.apk"
	outputDir := currentDesk + "\\funckapk\\output"
	channelPath := currentDesk + "\\funckapk\\bjxRecruit\\360Channel.txt"
	// 多渠道标识符列表
	channelIDs, err := util.ReadChannelIDs(channelPath)
	if err != nil {
		fmt.Printf("未找到渠道配置信息: %v\n", err)
		return
	}
	apkName := strings.Split(util.GetFileName(apkFilePath), ".apk")[0]
	fmt.Printf("老文件名: %v\n", apkName)

	// 临时目录用于反编译和重打包
	tempDir := outputDir + "\\tempApk"
	if !util.FileExists(apkFilePath) {
		fmt.Printf("母体apk文件未找到: %v\n", err)
		return
	}
	// 使用apktool反编译APK
	if err := runCommand("apktool", "d", apkFilePath, "-o", tempDir); err != nil {
		fmt.Printf("APK反编译过程中出错: %v\n", err)
		return
	}
	fmt.Printf("APK反编译成功\n")

	// 创建一个等待组，用于等待所有打包完成
	var wg sync.WaitGroup
	results := make(chan string)

	// 循环遍历渠道标识符列表，并分别进行打包
	for _, channelId := range channelIDs {
		wg.Add(1)
		go func(channel string) {
			defer wg.Done()
			result := PackAPK(tempDir, outputDir, apkName, channel)
			results <- result
		}(channelId)
	}
	//等待所有打包完成
	go func() {
		wg.Wait()
		close(results)
	}()

	//处理打包结果
	for result := range results {
		fmt.Printf("处理结果->%s\n", result)
	}
	// 删除临时目录
	err = os.RemoveAll(tempDir)
	if err != nil {
		fmt.Printf("删除临时文件时出错: %v\n", err)
		return
	}
	endTime := time.Now()                 // 记录结束时间
	elapsedTime := endTime.Sub(startTime) // 计算时间差
	fmt.Printf("全部渠道处理完成！总计用时：%s", elapsedTime)
}

// PackAPK 多渠道打包
func PackAPK(tempDir, outputDir, apkName, channelIDs string) string {
	parts := strings.Split(channelIDs, ",")
	channelKey := parts[0]
	channelID := parts[1]

	// 在当前目录下创建新文件夹
	newFolderPath := filepath.Join(outputDir, "tempAp-"+channelID)
	errCopy := os.Mkdir(newFolderPath, 0755)
	if errCopy != nil {
		fmt.Println("新建文件夹失败:", errCopy)
		return "新建文件夹失败"
	}
	//复制反编译结果文件
	err1 := util.CopyFolderContents(tempDir, newFolderPath)
	if err1 != nil {
		fmt.Println("复制反编译结果文件失败:", err1)
		return "复制反编译结果文件失败"
	}
	//开始处理
	fmt.Printf("正在处理处理 [%s] 渠道\n", channelID)
	// 修改AndroidManifest.xml文件中的渠道标识符
	if err := modifyManifestFile(tempDir, channelID, channelKey); err != nil {
		fmt.Printf("修改AndroidManifest.xml时出错: %v\n", err)
		return ""
	}
	newApkName := fmt.Sprintf("%s-%s", apkName, channelID)
	// 使用apktool重打包APK
	outputAPKPath := fmt.Sprintf("%s\\%s.apk", outputDir, newApkName)
	if err := runCommand("apktool", "b", tempDir, "-o", outputAPKPath); err != nil {
		fmt.Printf("APK重新打包过程中出错: %v\n", err)
		return ""
	}
	// 删除临时目录
	err := os.RemoveAll(newFolderPath)
	if err != nil {
		fmt.Printf("删除临时文件时出错: %v\n", err)
		return "删除临时文件时出错"
	}
	// 输出打包结果
	fmt.Printf("[%s]渠道打包完成...\n", channelID)
	fmt.Printf("新文件路径->[%s]...\n", outputAPKPath)
	return fmt.Sprintf("APK packed for channel: %s", channelID)
}

// 运行命令行命令的辅助函数
func runCommand(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// 修改AndroidManifest.xml文件中的渠道标识符
func modifyManifestFile(tempDir, channelID, channelKey string) error {
	manifestPath := tempDir + "\\AndroidManifest.xml"
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	// 将渠道标识符替换成目标渠道
	newContent := bytes.ReplaceAll(content, []byte(channelKey), []byte(channelID))
	if err := os.WriteFile(manifestPath, newContent, 0644); err != nil {
		return err
	}
	return nil
}
