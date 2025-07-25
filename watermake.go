package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// 支持的图片文件扩展名
var supportedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".bmp":  true,
	".gif":  true,
	".tiff": true,
	".webp": true,
}

// 获取系统字体路径
func getSystemFont() string {
	// 优先使用自定义字体路径（请替换为你的字体文件实际路径）
	customFontPath := "./SimHei.ttf" // macOS示例
	// customFontPath := "C:/fonts/simhei.ttf" // Windows示例
	if _, err := os.Stat(customFontPath); err == nil {
		return customFontPath
	}

	// 尝试常见的系统字体路径
	fontPaths := []string{
		"/System/Library/Fonts/PingFang.ttc",              // macOS 苹方
		"/Library/Fonts/Arial.ttf",                        // macOS Arial
		"/System/Library/Fonts/Helvetica.ttc",             // macOS Helvetica
		"C:/Windows/Fonts/simhei.ttf",                     // Windows 黑体
		"C:/Windows/Fonts/arial.ttf",                      // Windows Arial
		"/usr/share/fonts/truetype/freefont/FreeSans.ttf", // Linux
	}

	for _, path := range fontPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// 判断是否为支持的图片文件
func isImageFile(filename string) bool {
	ext := filepath.Ext(filename)
	_, ok := supportedExtensions[ext]
	return ok
}

// 添加水印
func addWatermark(imagePath, watermarkText, outputPath string,
	position string, opacity, fontSize int,
	randomColor bool, shadowOffset [2]int, shadowOpacity int) error {

	// 读取原始图片
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return fmt.Errorf("无法读取图片: %v", err)
	}

	// 处理WebP格式，转换为PNG以便处理
	ext := filepath.Ext(imagePath)
	if ext == ".webp" {
		// WebP格式处理逻辑已移除
		return fmt.Errorf("暂时不支持WebP格式")
	}

	// 解码图片
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("解码图片失败: %v", err)
	}

	// 创建绘图上下文
	dc := gg.NewContextForImage(img)
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	// 加载字体
	fontPath := getSystemFont()
	var face font.Face
	if fontPath != "" {
		fontData, err := os.ReadFile(fontPath)
		if err == nil {
			ttfFont, err := sfnt.Parse(fontData)
			if err == nil {
				face, err = opentype.NewFace(ttfFont, &opentype.FaceOptions{
					Size: float64(fontSize),
					DPI:  72,
				})
			}
		}
	}

	// 如果加载字体失败，使用备用方案
	if face == nil {
		fmt.Println("警告: 无法加载指定字体，使用备用字体")
		// 尝试使用gg库的默认字体加载机制
		if err := dc.LoadFontFace("sans-serif", float64(fontSize)); err != nil {
			// 如果仍然失败，使用内置字体
			fmt.Println("警告: 无法加载备用字体，使用内置字体")
		}
	} else {
		dc.SetFontFace(face)
	}

	// 计算文本尺寸
	textWidth, textHeight := dc.MeasureString(watermarkText)

	// 计算水印位置
	margin := 10
	var x, y float64

	switch position {
	case "top-left":
		x, y = float64(margin), float64(margin)
	case "top-right":
		x = float64(width) - textWidth - float64(margin)
		y = float64(margin)
	case "bottom-left":
		x = float64(margin)
		y = float64(height) - textHeight - float64(margin)
	case "center":
		x = (float64(width) - textWidth) / 2
		y = (float64(height) - textHeight) / 2
	default: // bottom-right
		x = float64(width) - textWidth - float64(margin)
		y = float64(height) - textHeight - float64(margin)
	}

	// 确定水印颜色
	var r, g, b uint8
	if randomColor {
		r = uint8(randomGenerator.Intn(256))
		g = uint8(randomGenerator.Intn(256))
		b = uint8(randomGenerator.Intn(256))
	} else {
		r, g, b = 255, 255, 255 // 默认白色
	}

	// 设置阴影颜色（比文字颜色深一些）
	shadowR := max(0, int(r)-100)
	shadowG := max(0, int(g)-100)
	shadowB := max(0, int(b)-100)

	// 绘制阴影
	dc.SetColor(color.RGBA{uint8(shadowR), uint8(shadowG), uint8(shadowB), uint8(shadowOpacity)})
	dc.DrawString(watermarkText, x+float64(shadowOffset[0]), y+float64(shadowOffset[1]))

	// 绘制文字
	dc.SetColor(color.RGBA{r, g, b, uint8(opacity)})
	dc.DrawString(watermarkText, x, y)

	// 确定输出路径和格式
	if outputPath == "" {
		dirName, fileName := filepath.Split(imagePath)
		name := fileName[:len(fileName)-len(filepath.Ext(fileName))]

		if ext == ".webp" {
			outputPath = filepath.Join(dirName, name+"_watermark.jpg")
		} else {
			outputPath = filepath.Join(dirName, name+"_watermark"+ext)
		}
	} else if filepath.Ext(outputPath) == ".webp" {
		outputPath = outputPath[:len(outputPath)-5] + ".jpg"
	}

	// 创建输出目录
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("无法创建输出目录: %v", err)
	}

	// 保存图片
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("无法创建输出文件: %v", err)
	}
	defer outputFile.Close()

	// 根据扩展名选择保存格式
	outputExt := filepath.Ext(outputPath)
	if outputExt == ".png" {
		if err := png.Encode(outputFile, dc.Image()); err != nil {
			return fmt.Errorf("保存PNG失败: %v", err)
		}
	} else { // 默认保存为JPG
		if err := jpeg.Encode(outputFile, dc.Image(), &jpeg.Options{Quality: 90}); err != nil {
			return fmt.Errorf("保存JPG失败: %v", err)
		}
	}

	fmt.Printf("已处理: %s\n", outputPath)
	return nil
}

// 批量处理目录
func processDirectory(inputDir, watermarkText, outputDir string,
	position string, opacity, fontSize int,
	randomColor bool, shadowOffset [2]int, shadowOpacity int) error {

	// 获取所有图片文件
	var imageFiles []string
	err := filepath.WalkDir(inputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && isImageFile(d.Name()) {
			imageFiles = append(imageFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("遍历目录失败: %v", err)
	}

	// 按创建时间排序
	sort.Slice(imageFiles, func(i, j int) bool {
		infoI, _ := os.Stat(imageFiles[i])
		infoJ, _ := os.Stat(imageFiles[j])
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	// 确定输出目录
	if outputDir == "" {
		outputDir = filepath.Join(inputDir, "watermarked")
		fmt.Printf("调试: 尝试创建输出目录: %s\n", outputDir)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("错误: 创建输出目录失败: %v\n", err)
			return fmt.Errorf("创建输出目录失败: %v", err)
		}
		fmt.Printf("将输出文件保存到: %s\n", outputDir)
	}

	// 处理每个图片
	for i, filePath := range imageFiles {
		filename := filepath.Base(filePath)
		ext := filepath.Ext(filename)

		// 生成序号文件名
		var outputFilename string
		if ext == ".webp" {
			outputFilename = strconv.Itoa(i+1) + ".jpg"
		} else {
			outputFilename = strconv.Itoa(i+1) + ext
		}

		outputPath := filepath.Join(outputDir, outputFilename)

		// 添加水印
		if err := addWatermark(filePath, watermarkText, outputPath,
			position, opacity, fontSize, randomColor, shadowOffset, shadowOpacity); err != nil {
			fmt.Printf("处理 %s 时出错: %v\n", filePath, err)
		}
	}

	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var randomGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))

func main() {
	// 打印所有命令行参数
	fmt.Printf("调试: 所有命令行参数: %v\n", os.Args)

	// 初始化默认值
	text := "Watermark"
	output := ""
	position := "bottom-right"
	opacity := 128
	size := 30
	noRandomColor := false
	shadowOffsetX := 2
	shadowOffsetY := 2
	shadowOpacity := 100

	// 手动解析命令行参数
	inputPath := ""
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-t" && i+1 < len(os.Args) {
			text = os.Args[i+1]
			i++
		} else if os.Args[i] == "-o" && i+1 < len(os.Args) {
			output = os.Args[i+1]
			i++
		} else if os.Args[i] == "-p" && i+1 < len(os.Args) {
			position = os.Args[i+1]
			i++
		} else if os.Args[i] == "-a" && i+1 < len(os.Args) {
			// 转换为整数
			if val, err := strconv.Atoi(os.Args[i+1]); err == nil {
				opacity = val
			}
			i++
		} else if os.Args[i] == "-s" && i+1 < len(os.Args) {
			// 转换为整数
			if val, err := strconv.Atoi(os.Args[i+1]); err == nil {
				size = val
			}
			i++
		} else if os.Args[i] == "-n" {
			noRandomColor = true
		} else if os.Args[i] == "-sox" && i+1 < len(os.Args) {
			// 转换为整数
			if val, err := strconv.Atoi(os.Args[i+1]); err == nil {
				shadowOffsetX = val
			}
			i++
		} else if os.Args[i] == "-soy" && i+1 < len(os.Args) {
			// 转换为整数
			if val, err := strconv.Atoi(os.Args[i+1]); err == nil {
				shadowOffsetY = val
			}
			i++
		} else if os.Args[i] == "-sa" && i+1 < len(os.Args) {
			// 转换为整数
			if val, err := strconv.Atoi(os.Args[i+1]); err == nil {
				shadowOpacity = val
			}
			i++
		} else if inputPath == "" {
			inputPath = os.Args[i]
		}
	}

	// 调试信息：输出参数值
	fmt.Printf("调试: -t 参数值: %s\n", text)

	// 检查输入路径
	if inputPath == "" {
		fmt.Println("请指定输入图片路径或文件夹路径")
		// 显示使用帮助
		fmt.Println("使用方法:")
		fmt.Println("  watermark [输入路径] -t [水印文字] -o [输出路径] -p [水印位置] -a [透明度] -s [字体大小] -n [不使用随机颜色] -sox [阴影X偏移] -soy [阴影Y偏移] -sa [阴影透明度]")
		fmt.Println("  水印位置: top-left, top-right, bottom-left, bottom-right, center")
		fmt.Println("  透明度范围: 0-255")
		return
	}

	if inputPath == "" {
		fmt.Println("请指定输入图片路径或文件夹路径")
		flag.Usage()
		return
	}

	// 验证参数
	if opacity < 0 || opacity > 255 {
		fmt.Println("透明度必须在0-255之间")
		return
	}

	if shadowOpacity < 0 || shadowOpacity > 255 {
		fmt.Println("阴影透明度必须在0-255之间")
		return
	}

	if size <= 0 {
		fmt.Println("字体大小必须大于0")
		return
	}

	// 检查位置参数有效性
	validPositions := map[string]bool{
		"top-left":     true,
		"top-right":    true,
		"bottom-left":  true,
		"bottom-right": true,
		"center":       true,
	}
	if !validPositions[position] {
		fmt.Println("无效的位置参数")
		return
	}

	// 判断输入路径类型
	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		fmt.Printf("错误: 无法访问 %s - %v\n", inputPath, err)
		return
	}

	shadowOffset := [2]int{shadowOffsetX, shadowOffsetY}
	randomColor := !noRandomColor

	if fileInfo.IsDir() {
		// 处理目录
		if err := processDirectory(inputPath, text, output,
			position, opacity, size, randomColor, shadowOffset, shadowOpacity); err != nil {
			fmt.Printf("处理目录时出错: %v\n", err)
		}
	} else {
		// 处理单个文件
		if !isImageFile(inputPath) {
			fmt.Println("错误: 输入文件不是支持的图片格式")
			return
		}

		if err := addWatermark(inputPath, text, output,
			position, opacity, size, randomColor, shadowOffset, shadowOpacity); err != nil {
			fmt.Printf("处理文件时出错: %v\n", err)
		}
	}
}
