package main

import (
	"apk-packer/util"
	"bufio"
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
	fmt.Printf("开始运行，正在读取本次任务需要的文件（母体apk、签名文件、渠道配置文件）...\n")
	// 在当前目录下创建新文件夹
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("获取当前文件夹失败...\n")
		return
	}
	// 获取当前目录下的所有文件
	files, err := util.GetFilesInDir(currentDir)
	if err != nil {
		fmt.Printf("获取文件失败...:\n")
		return
	}

	apkFiles := util.FilterApkFiles(files)
	if len(apkFiles) == 0 {
		fmt.Printf("没有找到Apk文件...:\n")
		return
	}
	channelFiles := util.FilterTxtFiles(files)
	if len(channelFiles) == 0 {
		fmt.Printf("没有找到渠道配置文件...:")
		return
	}
	jksFiles := util.FilterJksFiles(files)
	if len(jksFiles) == 0 {
		fmt.Printf("没有找到签名文件...:")
		return
	}

	apkFilePath := apkFiles[0]
	outputDir := currentDir + "\\output"
	channelPath := channelFiles[0]
	jksPath := jksFiles[0]

	// 多渠道标识符列表
	channelIDs, err := util.ReadChannelIDs(channelPath)
	if err != nil {
		fmt.Printf("没有找到渠道配置信息: %v\n", err)
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
	fmt.Printf("读取完毕，开始执行反编译APK...\n")
	// 使用apktool反编译APK
	if err := runCommand(true, "apktool", "d", apkFilePath, "-o", tempDir); err != nil {
		fmt.Printf("APK反编译过程中出错: %v\n", err)
		return
	}
	fmt.Printf("APK反编译成功!\n")
	fmt.Printf("下面开始执行多渠道打包...!\n")
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
	fmt.Printf("全部渠道打包完成！总计用时：%s\n", elapsedTime)
	fmt.Printf("正在重新签名...\n")
	// 获取当前目录下的所有文件
	outFiles, err := util.GetFilesInDir(outputDir)
	if err != nil {
		fmt.Printf("获取文件失败...:\n")
		return
	}
	allApks := util.FilterApkFiles(outFiles)
	signAPKsWithJks(allApks, jksPath)
	endTime2 := time.Now()                // 记录结束时间
	elapsedTime2 := endTime2.Sub(endTime) // 计算时间差
	fmt.Printf("全部签名完成！总计用时：%s", elapsedTime2)
	// 等待用户输入任意键退出
	fmt.Println("按下任意键退出...")
	_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
}

// PackAPK 多渠道打包
func PackAPK(tempDir, outputDir, apkName, channelIDs string) string {
	parts := strings.Split(channelIDs, ",")
	channelKey := parts[0]
	channelID := parts[1]

	// 在当前目录下创建新文件夹
	newFolderPath := filepath.Join(outputDir, "tempApk-"+channelID)
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
	fmt.Printf("开始处理 [%s] 渠道\n", channelID)
	// 修改AndroidManifest.xml文件中的渠道标识符
	if err := modifyManifestFile(newFolderPath, channelID, channelKey); err != nil {
		fmt.Printf("修改AndroidManifest.xml时出错: %v\n", err)
		return fmt.Sprintf("修改AndroidManifest.xml时出错: %v\n", err)
	}
	newApkName := fmt.Sprintf("%s-%s", apkName, channelID)
	// 使用apktool重打包APK
	outputAPKPath := fmt.Sprintf("%s\\%s.apk", outputDir, newApkName)
	cmd := exec.Command("apktool", "b", newFolderPath, "-o", outputAPKPath)
	output, runErr := cmd.CombinedOutput()
	if runErr != nil {
		fmt.Printf("APK重新打包过程中出错: %v\n", runErr)
		return fmt.Sprintf("APK重新打包过程中出错: %v\n", runErr)
	}
	// 删除临时目录
	err := os.RemoveAll(newFolderPath)
	if err != nil {
		fmt.Printf("删除临时文件时出错: %v\n", err)
		return "删除临时文件时出错"
	}
	// 输出打包结果
	fmt.Printf("[%s]%s渠道打包完成...\n", channelID, output)
	return fmt.Sprintf("[%s]%s渠道打包完成...\n", channelID, output)
}

func runCommand(needLog bool, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if needLog {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
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

func signAPKsWithJks(apkFiles []string, jksPath string) {
	for _, apkFile := range apkFiles {
		// 签名文件信息
		keyAlias := util.GetAliasName(jksPath)
		keyPassword := util.GetKeyPassword(jksPath)
		storePassword := util.GetStorePassword(jksPath)

		// 签名APK
		cmd := exec.Command("jarsigner", "-verbose", "-sigalg", "SHA1withRSA", "-digestalg", "SHA1", "-keystore", jksPath, "-storepass", storePassword, "-keypass", keyPassword, apkFile, keyAlias)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("签名失败：%s: %v\nOutput:\n%s\n", apkFile, err, string(output))
		} else {
			fmt.Printf("APK %s.apk 签名成功.\n", util.GetFileName(apkFile))
		}
	}
}
