package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gen2brain/go-unarr"
)

///// struct
type Setting struct {
	Version       string
	LatestVersion string
	LocalVersion  string
	Owner         string
	Repo          string
	ThisOwner     string
	ThisRepo      string
	API           []string
}

type Asset struct {
	URL                string `json:"url"`
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	ContentType        string `json:"content_type"`
	State              string `json:"state"`
	Size               int    `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type Latest struct {
	URL     string  `json:"url"`
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Message string  `json:"message"`
	Assets  []Asset `json:"assets"`
}

///// until
//打开文件和读内容 利用io/ioutil
func readAll(path string) (string, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	//对内容的操作
	//ReadFile返回的是[]byte字节切片，要用string()方法转变成字符串
	//去除内容结尾的换行符
	str := strings.TrimRight(string(content), "\n")
	return str, nil
}

//文件写入 先清空再写入 利用ioutil
func writeFast(filePath string, content string) error {
	dir, _ := path.Split(filePath)
	exist, err := isFileExisted(dir)
	if err != nil {
		return err
	} else if exist == false {
		os.Mkdir(dir, os.ModePerm)
	}
	err = ioutil.WriteFile(filePath, []byte(content), 0666)
	if err != nil {
		return err
	} else {
		return nil
	}
}

//判断文件/文件夹是否存在
func isFileExisted(path string) (bool, error) {
	//返回 true, nil = 存在
	//返回 false, nil = 不存在
	//返回 _, !nil = 位置错误，无法判断
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

//利用HTTP Get请求获得数据
func getHttpData(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	_ = resp.Body.Close()

	return string(data), nil
}

//下载文件 (下载地址，存放位置)
func downloadFile(url string, location string) error {
	//利用HTTP下载文件并读取内容给data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		errorInfo := "http failed, check if file exists, HTTP Status Code:" + strconv.Itoa(resp.StatusCode)
		return errors.New(errorInfo)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	//确保下载位置存在
	_, fileName := path.Split(url)
	ok, err := isFileExisted(location)
	if err != nil {
		return err
	} else if ok == false {
		err := os.Mkdir(location, os.ModePerm)
		if err != nil {
			return err
		}
	}
	//文件写入 先清空再写入 利用ioutil
	err = ioutil.WriteFile(location+"/"+fileName, data, 0666)
	if err != nil {
		return err
	} else {
		return nil
	}
}

//压缩
func Zip(from string, toZip string) error {
	zipfile, err := os.Create(toZip)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	_ = filepath.Walk(from, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = strings.TrimPrefix(path, filepath.Dir(from)+"/")
		// header.Name = path
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
		}
		return err
	})

	return err
}

//解压
func Unzip(zipFile string, to string) error {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	for _, f := range zipReader.File {
		fpath := filepath.Join(to, f.Name)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(fpath, os.ModePerm)
		} else {
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}

			inFile, err := f.Open()
			defer inFile.Close()
			if err != nil {
				return err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			defer outFile.Close()
			if err != nil {
				return err
			}

			_, err = io.Copy(outFile, inFile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//解压zip 7z rar tar
func decompress(from string, to string) error {
	a, err := unarr.NewArchive(from)
	if err != nil {
		return err
	}
	defer a.Close()

	_, err = a.Extract(to)
	if err != nil {
		return err
	}

	return nil
}

///// important functions
//解析Json，获取最新版本号和下载地址 return TagName, Asset Slice, nil
func parseReleaseInfo(owner string, repo string) (string, []Asset, error) {
	//GET请求获得JSON
	jsonData, err := getHttpData("https://api.github.com/repos/" + owner + "/" + repo + "/releases/latest")
	if err != nil {
		log.Println(err)
		return "", nil, err
	}

	//初始化实例并解析JSON
	var latestInst Latest
	err = json.Unmarshal([]byte(jsonData), &latestInst) //第二个参数要地址传递
	if err != nil {
		return "", nil, err
	}

	//链接有问题也会返回Json，且 "Message": "Not Found"
	if latestInst.Message == "Not Found" {
		return "", nil, errors.New("got Json but no valid. Check URL")
	}

	return latestInst.TagName, latestInst.Assets, nil
}

func readSettings(path string) (Setting, error) {
	//检查文件是否存在
	exist, err := isFileExisted(path)
	if err != nil {
		return Setting{}, err
	} else if exist == true {
		//存在则读取文件
		content, err := readAll(path)
		if err != nil {
			return Setting{}, err
		}

		//初始化实例并解析JSON
		var settingInst Setting
		err = json.Unmarshal([]byte(content), &settingInst) //第二个参数要地址传递
		if err != nil {
			return Setting{}, err
		}

		var transporter = Setting{
			Version:       "0.1.4",
			LatestVersion: "",
			LocalVersion:  "",
			Owner:         "advancedfx",
			Repo:          "advancedfx",
			ThisOwner:     "Purple-CSGO",
			ThisRepo:      "HLAE-Release",
			API:           []string{},
		}
		return transporter, nil
	}

	return Setting{}, err
}

func saveSettings(path string) error {
	//检查文件是否存在
	exist, err := isFileExisted(path)
	if err != nil {
		return err
	} else if exist == true {
		//存在则读取文件
		content, err := readAll(path)
		if err != nil {
			return err
		}

		var settingInst Setting
		err = json.Unmarshal([]byte(content), &settingInst) //第二个参数要地址传递
		if err != nil {
			return err
		}

	}

	return nil
}

func main() {
	//1.读取settings.json，不存在或出错则赋默认值
	Transporter := &Setting{
		Version:       "0.1.0",
		LatestVersion: "",
		LocalVersion:  "",
		Owner:         "advancedfx",
		Repo:          "advancedfx",
		ThisOwner:     "Purple-CSGO",
		ThisRepo:      "HLAE-Release",
		API:           []string{},
	}

	temp, err := readSettings("./settings.json")
	if err != nil {
		log.Fatal(err)
	} else {
		Transporter = &temp
	}

	//2.Welcome~
	fmt.Printf(Transporter.Version)

	//3.如果本地版本为空，利用API获取包含该仓库信息的JSON文件并解析，获得版本号
	fmt.Println("正在获取", Transporter.ThisOwner, '/', Transporter.ThisRepo, "最新版本信息...")
	tagName, _, err := parseReleaseInfo(Transporter.ThisOwner, Transporter.ThisRepo)
	if err != nil {
		log.Fatal(err)
	} else {
		Transporter.LocalVersion = tagName
	}

	//4.利用API获取包含仓库信息的JSON文件并解析，获得版本号和附件切片
	fmt.Println("正在获取", Transporter.Owner, '/', Transporter.Repo, "最新版本信息...")
	tagName, assets, err := parseReleaseInfo(Transporter.Owner, Transporter.Repo)
	if err != nil {
		log.Fatal(err)
	} else {
		Transporter.LatestVersion = tagName
	}

	//5.处理附件切片 TODO:可选各种处理方法 下载/改名/根目录或者'./$tagName'
	for _, asset := range assets {
		fmt.Println(asset.Name)
		//获得下载链接
		//var url, fileName string
		//for _, file := range latestInst.Assets {
		//	//过滤掉源码文件
		//	if file.State == "uploaded" && !strings.Contains(file.Name, ".asc") && strings.Contains(file.Name, ".zip") {
		//		url = file.BrowserDownloadURL
		//		fileName = file.Name
		//	}
		//}
	}

}
