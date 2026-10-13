package knowledge

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

const (
	maxFileBytes = 10 * 1024 * 1024 // 单个导入文件大小上限
	maxXMLBytes  = 20 * 1024 * 1024 // docx 内 document.xml 解压后上限（防压缩炸弹）
)

// IsPDF 文件是否为 PDF（需由模型解析）
func IsPDF(path string) bool { return strings.EqualFold(filepath.Ext(path), ".pdf") }

// ExtractLocal 在本地提取 .docx / .md / .markdown / .txt 的文本；PDF 不在此处理。
func ExtractLocal(path string) (string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".txt":
		data, err := readLimited(path)
		if err != nil {
			return "", err
		}
		return decodeText(data), nil
	case ".docx":
		data, err := readLimited(path)
		if err != nil {
			return "", err
		}
		return extractDocx(data)
	default:
		return "", fmt.Errorf("不支持的文件类型 %q，请选择 PDF / DOCX / MD / TXT", filepath.Ext(path))
	}
}

func readLimited(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxFileBytes {
		return nil, fmt.Errorf("文件超过 %dMB 限制", maxFileBytes/1024/1024)
	}
	return data, nil
}

// decodeText 优先按 UTF-8 解读；不是合法 UTF-8 时按 GBK 回退（国内常见的 txt 编码）
func decodeText(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}
	if out, err := simplifiedchinese.GBK.NewDecoder().Bytes(data); err == nil {
		return string(out)
	}
	return strings.ToValidUTF8(string(data), "")
}

// ==================== DOCX ====================

var headingStyleRe = regexp.MustCompile(`(?i)^(?:heading|标题)\s*([1-6])$`)

// extractDocx 从 .docx 的 word/document.xml 提取文本：
// 段落按行输出，标题样式转为 Markdown 标题，列表项转为 "- "，表格每行用 " | " 连接单元格。
func extractDocx(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", errors.New("无法打开 DOCX 文件（可能已损坏，或是旧版 .doc 格式）")
	}
	var doc *zip.File
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			doc = f
			break
		}
	}
	if doc == nil {
		return "", errors.New("DOCX 中未找到正文（word/document.xml）")
	}
	rc, err := doc.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	dec := xml.NewDecoder(io.LimitReader(rc, maxXMLBytes))
	var (
		out       strings.Builder
		para      strings.Builder // 当前段落文本
		heading   int             // 当前段落标题级别（0 = 非标题）
		isList    bool
		inCell    bool
		rowCells  []string
		cellParas []string
		tblDepth  int
	)

	flushPara := func() {
		text := strings.TrimSpace(para.String())
		para.Reset()
		level, list := heading, isList
		heading, isList = 0, false
		if text == "" {
			return
		}
		switch {
		case level > 0:
			text = strings.Repeat("#", level) + " " + text
		case list:
			text = "- " + text
		}
		if inCell {
			cellParas = append(cellParas, text)
			return
		}
		out.WriteString(text)
		out.WriteString("\n\n")
	}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("解析 DOCX 失败: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Space != "" && !strings.HasSuffix(t.Name.Space, "wordprocessingml/2006/main") {
				continue
			}
			switch t.Name.Local {
			case "tbl":
				tblDepth++
			case "tc":
				inCell = true
				cellParas = cellParas[:0]
			case "pStyle":
				for _, a := range t.Attr {
					if a.Name.Local == "val" {
						if m := headingStyleRe.FindStringSubmatch(a.Value); m != nil {
							heading = int(m[1][0] - '0')
						}
					}
				}
			case "numPr":
				isList = true
			case "tab":
				para.WriteString("\t")
			case "br", "cr":
				para.WriteString("\n")
			case "t":
				var text string
				if err := dec.DecodeElement(&text, &t); err == nil {
					para.WriteString(text)
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "p":
				flushPara()
			case "tc":
				rowCells = append(rowCells, strings.Join(cellParas, " "))
				cellParas = cellParas[:0]
				inCell = false
			case "tr":
				if line := strings.TrimSpace(strings.Join(rowCells, " | ")); strings.Trim(line, "| ") != "" {
					out.WriteString(line)
					out.WriteString("\n")
				}
				rowCells = rowCells[:0]
			case "tbl":
				if tblDepth > 0 {
					tblDepth--
				}
				out.WriteString("\n")
			}
		}
	}
	text := Normalize(out.String())
	if text == "" {
		return "", errors.New("DOCX 中没有提取到文字（可能全是图片）")
	}
	return text, nil
}
