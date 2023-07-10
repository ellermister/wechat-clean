package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// 文件名列举
// f4ae4a01ae1a5c98653398c7472ec695.temp.jpg
// th_f4ae2fd19a1dc35077a099cc04c0fb38
// th_f4ae4a01ae1a5c98653398c7472ec695
// th_f4ae4a01ae1a5c98653398c7472ec695hd
// th_f4aec2737a56ceba61fb522ecccc0208
// ./0c/1f/0c1f25bf5860433903600373b33aaa90.temp.jpg_hevc
// ./ae/e6/aee6942e3807422bd076d63f5f04b903.temp

// FindImagesByID
// 返回的是 /image2/ae/e6/aee6942e3807422bd076d63f5f04b903.temp 之类的格式 或者绝对路径

func FindImagesByID(userPath string, idStr string, isAbsPath bool) []string {
	if len(idStr) != 32 {
		return []string{}
	}

	seg1 := idStr[:2]
	seg2 := idStr[2:4]

	var image2Dir = "/image2" + "/" + seg1 + "/" + seg2 + "/"
	var prefix = userPath + image2Dir
	files1, _ := filepath.Glob(prefix + idStr + "*")
	files2, _ := filepath.Glob(prefix + "th_" + idStr + "*")

	// Glob 返回的就是给定的参数的起始路径，所以是完整的
	files := append(files1, files2...)

	if isAbsPath {
		return files
	} else {
		var filenames []string
		for _, value := range files {
			filenames = append(filenames, image2Dir+filepath.Base(value))
		}
		return filenames
	}

}

func FindVideosByID(userPath string, idStr string, isAbsPath bool) []string {
	// 过滤掉空及非法字符
	if len(idStr) < 4 {
		return []string{}
	}
	files, _ := filepath.Glob(userPath + "/video/" + idStr + "*")

	if isAbsPath {
		return files
	} else {
		var filenames []string
		for _, value := range files {
			filenames = append(filenames, "/video/"+filepath.Base(value))
		}
		return filenames
	}

}

func StatFileSize(absPath string) int64 {
	if Exists(absPath) {
		state, _ := os.Stat(absPath)
		return state.Size()
	}
	return int64(0)
}

// 获取目录dir下的文件名字
func walkDir(dir string, wg *sync.WaitGroup, files chan<- os.FileInfo) {
	defer wg.Done()
	for _, entry := range dirents(dir) {
		if entry.IsDir() { //目录
			wg.Add(1)
			subDir := filepath.Join(dir, entry.Name())
			go walkDir(subDir, wg, files)
		} else {
			files <- entry
		}
	}
}

var sema = make(chan struct{}, 20)

// 读取目录dir下的文件信息
func dirents(dir string) []os.FileInfo {
	sema <- struct{}{}
	defer func() { <-sema }()
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "du: %v\n", err)
		return nil
	}
	return entries
}

func getDirFiles(dirPath string) []string {
	roots := []string{dirPath}
	filesInfo := make(chan os.FileInfo)

	var wg sync.WaitGroup
	for _, root := range roots {
		wg.Add(1)
		go walkDir(root, &wg, filesInfo)
	}
	go func() {
		wg.Wait() //等待goroutine结束
		close(filesInfo)
	}()

	var filenames []string
loop:
	for {
		select {
		case thefile, ok := <-filesInfo:
			if !ok {
				break loop
			}
			filenames = append(filenames, thefile.Name())
		}
	}

	return filenames
}

type DirStatSize struct {
	nFiles int64
	nBytes int64
}

func getDirSize(dirPath string) DirStatSize {
	roots := []string{dirPath}
	filesInfo := make(chan os.FileInfo)

	var wg sync.WaitGroup
	for _, root := range roots {
		wg.Add(1)
		go walkDir(root, &wg, filesInfo)
	}
	go func() {
		wg.Wait()
		close(filesInfo)
	}()

	var _dirStat DirStatSize
loop:
	for {
		select {
		case theFile, ok := <-filesInfo:
			if !ok {
				break loop
			}
			_dirStat.nFiles++
			_dirStat.nBytes += theFile.Size()
		}
	}

	return _dirStat
}

func removeSubDirAndFiles(dirname string) {
	files, err := ioutil.ReadDir(dirname)
	if err != nil {
		log.Printf("removeSubDirAndFiles err:  %v", err)
		return
	}

	// 判断底下是否有文件
	if len(files) > 0 {
		for _, filename := range files {
			os.RemoveAll(dirname + "/" + filename.Name())
		}
	}
}
