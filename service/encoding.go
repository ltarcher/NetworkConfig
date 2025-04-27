package service

import (
	"bytes"
	"io/ioutil"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// DecodeToUTF8 尝试将可能的GBK编码转换为UTF-8
func DecodeToUTF8(input []byte) ([]byte, error) {
	// 如果输入已经是有效的UTF-8，直接返回
	if isUTF8(input) {
		return input, nil
	}

	// 尝试从GBK转换到UTF-8
	reader := transform.NewReader(bytes.NewReader(input), simplifiedchinese.GBK.NewDecoder())
	decoded, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

// isUTF8 检查输入是否为有效的UTF-8编码
func isUTF8(input []byte) bool {
	return utf8.Valid(input)
}
