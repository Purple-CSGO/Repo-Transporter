@echo off
chcp 65001

:: Windows环境下编译可执行文件
SET GOOS=linux
SET GOARCH=amd64
go build -o ./build/ main.go

echo ---------------------- Start ----------------------

:: 运行可执行文件 文件名务必更改
::cd build

::repo-transporter

echo ----------------------- End -----------------------
