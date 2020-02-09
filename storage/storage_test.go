package storage

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStorageDelete(t *testing.T) {
	InitStorage()
	Remove("midterm/Gopax/btc/export")
}

func TestPng2Bytes(t *testing.T) {
	aa := PNGtoBytes("../analyser/images/candle1.png")
	fmt.Println("Length: ", len(aa))
	dataType := http.DetectContentType(aa)
	fmt.Println("DataType: ", dataType)
}

func TestUploadImage(t *testing.T) {
	InitStorage()
	// Cleanup
	Remove("tmpday20200209")

	// Upload new images(today)
	err := filepath.Walk("../analyser/tmpday20200209", func(path string, info os.FileInfo, err error) error {
		png := PNGtoBytes(path)
		contentType := http.DetectContentType(png)
		if strings.HasSuffix(contentType, "png") {
			splits := strings.Split(path, "/")
			splits = splits[len(splits)-2:]
			savePath := strings.Join(splits, "/")
			Write(png, savePath)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}
