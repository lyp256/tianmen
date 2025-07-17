package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
)

func main() {
	stderrReader, stderrWriter := io.Pipe()
	defer stderrWriter.Close()

	// 1. 创建要执行的命令
	cmd := exec.Command("zsh") // 可以是 bash, ssh 等
	cmd.Stderr = stderrWriter

	// 2. 创建伪终端
	ptmx, err := pty.Start(cmd)
	if err != nil {
		panic("启动伪终端失败: " + err.Error())
	}
	defer ptmx.Close() // 确保关闭伪终端

	//// 3. 设置原始终端模式
	//oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	//if err != nil {
	//	panic("设置原始模式失败: " + err.Error())
	//}
	//defer term.Restore(int(os.Stdin.Fd()), oldState) // 程序退出时恢复终端

	// 4. 处理终端大小变化
	resizeTty := make(chan os.Signal, 1)
	signal.Notify(resizeTty, syscall.SIGWINCH)
	defer signal.Stop(resizeTty)

	go func() {
		for range resizeTty {
			if size, err := pty.GetsizeFull(os.Stdin); err == nil {
				pty.Setsize(ptmx, size)
			}
		}
	}()
	resizeTty <- syscall.SIGWINCH // 初始化大小

	// 5. 处理输入输出流
	go func() {
		io.Copy(ptmx, os.Stdin) // 将本地输入发送到伪终端
	}()

	// 6. 处理输出（带缓冲区刷新）
	go func() {
		io.CopyBuffer(os.Stdout, ptmx, make([]byte, 1024))
	}()

	go func() {
		io.CopyBuffer(os.Stderr, stderrReader, make([]byte, 1024))
	}()

	// 7. 等待命令结束
	if err := cmd.Wait(); err != nil {
		fmt.Println("命令执行错误:", err)
	}
}
