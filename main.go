package main

import (
	"apk-packer/util"
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed util/apk-packer.exe
var exeData []byte

func main() {
	tempDirr := os.TempDir()
	exePackageDir := filepath.Join(tempDirr, "TempApplication")
	_ = os.MkdirAll(exePackageDir, os.ModePerm)
	exePackageFile := filepath.Join(exePackageDir, "apk-packer.exe")
	targetFile, _ := os.Create(exePackageFile)
	_, _ = targetFile.Write(exeData)
	_ = targetFile.Close()
	cmd := exec.Command(exePackageFile)
	cmd.Start()
	go func(cmd *exec.Cmd) {
		_ = cmd.Wait()
	}(cmd)
	startTime := time.Now()
	fmt.Printf("开始运行，正在读取本次任务需要的文件（母体apk、签名文件、渠道配置文件）...\n")
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("获取当前文件夹失败...\n")
		return
	}
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
	//apkToolPath := currentDir + "\\ApkTool\\apktool.bat"
	apkToolPath := "apktool"
	channelIDs, err := util.ReadChannelIDs(channelPath)
	if err != nil {
		fmt.Printf("没有找到渠道配置信息: %v\n", err)
		return
	}
	apkName := strings.Split(util.GetFileName(apkFilePath), ".apk")[0]
	fmt.Printf("老文件名: %v\n", apkName)
	tempDir := outputDir + "\\tempApk"
	if !util.FileExists(apkFilePath) {
		fmt.Printf("母体apk文件未找到: %v\n", err)
		return
	}
	fmt.Printf("读取完毕，开始执行反编译APK...\n")
	if err := runCommand(true, apkToolPath, "d", apkFilePath, "-o", tempDir); err != nil {
		fmt.Printf("APK反编译过程中出错: %v\n", err)
		return
	}
	fmt.Printf("APK反编译成功!\n")
	fmt.Printf("下面开始执行多渠道打包...!\n")
	var wg sync.WaitGroup
	results := make(chan string)
	for _, channelId := range channelIDs {
		wg.Add(1)
		go func(channel string) {
			defer wg.Done()
			result := PackAPK(tempDir, outputDir, apkName, channel, apkToolPath)
			results <- result
		}(channelId)
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	for result := range results {
		fmt.Printf("处理结果->%s\n", result)
	}
	err = os.RemoveAll(tempDir)
	if err != nil {
		fmt.Printf("删除临时文件时出错: %v\n", err)
		return
	}
	endTime := time.Now()
	elapsedTime := endTime.Sub(startTime)
	fmt.Printf("全部渠道打包完成！总计用时：%s\n", elapsedTime)
	fmt.Printf("正在重新签名...\n")
	outFiles, err := util.GetFilesInDir(outputDir)
	if err != nil {
		fmt.Printf("获取文件失败...:\n")
		return
	}
	allApks := util.FilterApkFiles(outFiles)
	signAPKsWithJks(allApks, jksPath)
	endTime2 := time.Now()
	elapsedTime2 := endTime2.Sub(endTime)
	fmt.Printf("全部签名完成！总计用时：%s", elapsedTime2)
	fmt.Println("按下任意键退出...")
	_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
}

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
		fmt.Printf("已删除原ApkTemp文件\n")
	}
	errCopy := os.Mkdir(newFolderPath, 0755)
	if errCopy != nil {
		fmt.Println("新建文件夹失败:", errCopy)
		return "新建文件夹失败"
	}
	err1 := util.CopyFolderContents(tempDir, newFolderPath)
	if err1 != nil {
		fmt.Println("复制反编译结果文件失败:", err1)
		return "复制反编译结果文件失败"
	}
	fmt.Printf("开始处理 [%s] 渠道\n", channelID)
	if err := modifyManifestFile(newFolderPath, channelID, channelKey); err != nil {
		fmt.Printf("修改AndroidManifest.xml时出错: %v\n", err)
		return fmt.Sprintf("修改AndroidManifest.xml时出错: %v\n", err)
	}
	newApkName := fmt.Sprintf("%s-%s", apkName, channelID)
	outputAPKPath := fmt.Sprintf("%s\\%s.apk", outputDir, newApkName)
	cmd := exec.Command(apkToolPath, "b", newFolderPath, "-o", outputAPKPath)
	output, runErr := cmd.CombinedOutput()
	if runErr != nil {
		fmt.Printf("APK重新打包过程中出错: %v\n", runErr)
		return fmt.Sprintf("APK重新打包过程中出错: %v\n", runErr)
	}
	err := os.RemoveAll(newFolderPath)
	if err != nil {
		fmt.Printf("删除临时文件时出错: %v\n", err)
		return "删除临时文件时出错"
	}
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

func modifyManifestFile(tempDir, channelID, channelKey string) error {
	manifestPath := tempDir + "\\AndroidManifest.xml"
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	newContent := bytes.ReplaceAll(content, []byte(channelKey), []byte(channelID))
	if err := os.WriteFile(manifestPath, newContent, 0644); err != nil {
		return err
	}
	return nil
}

func signAPKsWithJks(apkFiles []string, jksPath string) {
	for _, apkFile := range apkFiles {
		keyAlias := util.GetAliasName(jksPath)
		keyPassword := util.GetKeyPassword(jksPath)
		storePassword := util.GetStorePassword(jksPath)
		cmd := exec.Command("jarsigner", "-verbose", "-sigalg", "SHA1withRSA", "-digestalg", "SHA1", "-keystore", jksPath, "-storepass", storePassword, "-keypass", keyPassword, apkFile, keyAlias)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("签名失败：%s: %v\nOutput:\n%s\n", apkFile, err, string(output))
		} else {
			fmt.Printf("APK %s 签名成功.\n", util.GetFileName(apkFile))
		}
	}
}
