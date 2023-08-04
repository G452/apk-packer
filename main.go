package main

import (
	"apk-packer/util"
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

//go:embed util/apk-packer.exe
var exeData []byte

func main() {
	//tempDirr := os.TempDir()
	//exePackageDir := filepath.Join(tempDirr, "TempApplication")
	//_ = os.MkdirAll(exePackageDir, os.ModePerm)
	//exePackageFile := filepath.Join(exePackageDir, "apk-packer.exe")
	//targetFile, _ := os.Create(exePackageFile)
	//_, _ = targetFile.Write(exeData)
	//_ = targetFile.Close()
	//cmd := exec.Command(exePackageFile)
	//cmd.Start()
	//go func(cmd *exec.Cmd) {
	//	_ = cmd.Wait()
	//}(cmd)
	startTime := time.Now()
	fmt.Printf("开始运行，正在读取本次任务需要的文件（目标apk、签名文件、渠道配置文件）...\n")
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("获取当前文件夹失败...\n")
		return
	}
	fmt.Printf("当前目录：%s\n", currentDir)
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
	fmt.Printf("目标apk：%s\n", util.GetFileName(apkFilePath))
	fmt.Printf("签名文件：%s\n", util.GetFileName(jksPath))
	fmt.Printf("渠道配置：%s\n", util.GetFileName(channelPath))
	fmt.Printf("输出路径：%s\n", outputDir)
	channelIDs, err := util.ReadChannelIDs(channelPath)
	if err != nil {
		fmt.Printf("没有找到渠道配置信息: %v\n", err)
		return
	}
	fmt.Printf("一共%d个渠道，开始处理...\n", len(channelIDs))
	apkName := strings.Split(util.GetFileName(apkFilePath), ".apk")[0]
	tempDir := outputDir + "\\tempApk"
	if !util.FileExists(apkFilePath) {
		fmt.Printf("母体apk文件未找到: %v\n", err)
		return
	}
	if _, err := os.Stat(tempDir); err == nil {
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Printf("删除文件失败: %s\n", err)
			return
		}
		fmt.Printf("删除原文件夹" + tempDir + "\n")
	}
	fmt.Printf("读取完毕，正在执行反编译[%s]...\n", util.GetFileName(apkFilePath))
	cmd := exec.Command(apkToolPath, "d", apkFilePath, "-o", tempDir)
	_, runErr := cmd.Run()
	if runErr != nil {
		fmt.Printf("APK反编译启动失败: %v\n", runErr)
		return
	}
	fmt.Printf("APK反编译成功!\n")
	fmt.Printf("正在执行多渠道打包...\n")
	var wg sync.WaitGroup
	results := make(chan string)
	for _, channelId := range channelIDs {
		wg.Add(1)
		go func(channel string) {
			defer wg.Done()
			result := util.PackAPK(tempDir, outputDir, apkName, channel, apkToolPath)
			results <- result
		}(channelId)
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	for result := range results {
		fmt.Printf(">%s", result)
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
	util.SignAPKsWithJks(allApks, jksPath)
	endTime2 := time.Now()
	elapsedTime2 := endTime2.Sub(endTime)
	fmt.Printf("全部签名完成！总计用时：%s\n", elapsedTime2)
	fmt.Printf("按下任意键退出...\n")
	_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
}
