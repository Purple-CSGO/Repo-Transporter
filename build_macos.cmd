@echo off
chcp 65001

:: 编译可执行文件
SET GOOS=darwin
SET GOARCH=amd64
go build -o ./build/repo-transporter main.go

echo ---------------------- Start ----------------------

:: 运行可执行文件 文件名务必更改
::cd build

::repo-transporter

echo ----------------------- End -----------------------
