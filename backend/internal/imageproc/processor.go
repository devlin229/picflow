package imageproc

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/HugoSmits86/nativewebp"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/webp"
)

// ProcessConfig 描述规则型图片处理参数。
type ProcessConfig struct {
	CanvasWidth         int    `json:"canvas_width"`
	CanvasHeight        int    `json:"canvas_height"`
	Background          string `json:"background"`
	LayoutMode          string `json:"layout_mode"`
	KeepSubjectComplete bool   `json:"keep_subject_complete"`
	OutputFormat        string `json:"output_format"`
	MarginMode          string `json:"margin_mode"`
	Margin              int    `json:"margin"`
}

// SizeConfig 描述尺寸图上的用户输入。
type SizeConfig struct {
	Width  float64
	Height float64
	Depth  float64
	Unit   string
}

// SizeTemplateConfig 描述尺寸图模板对画布和标注样式的控制。
type SizeTemplateConfig struct {
	CanvasWidth     int    `json:"canvas_width"`
	CanvasHeight    int    `json:"canvas_height"`
	Background      string `json:"background"`
	OutputFormat    string `json:"output_format"`
	Margin          int    `json:"margin"`
	AnnotationColor string `json:"annotation_color"`
}

// Inspect 读取图片真实尺寸与格式。
func Inspect(path string) (width, height int, format string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, "", err
	}
	defer file.Close()
	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, "", fmt.Errorf("无法解析图片: %w", err)
	}
	if format != "jpeg" && format != "png" && format != "webp" {
		return 0, 0, "", fmt.Errorf("仅支持 JPG、PNG 和 WebP")
	}
	return config.Width, config.Height, format, nil
}

// Standardize 按等比缩放和补边规则生成标准商品图。
func Standardize(sourcePath, outputPath string, config ProcessConfig) error {
	source, err := decode(sourcePath)
	if err != nil {
		return err
	}
	canvas, err := compose(source, config)
	if err != nil {
		return err
	}
	return encode(outputPath, canvas, config.OutputFormat)
}

// SizeChart 生成带宽、高和可选深度标注的尺寸图。
func SizeChart(sourcePath, outputPath string, size SizeConfig, template SizeTemplateConfig) error {
	source, err := decode(sourcePath)
	if err != nil {
		return err
	}
	canvas, err := compose(source, ProcessConfig{
		CanvasWidth: template.CanvasWidth, CanvasHeight: template.CanvasHeight, Background: template.Background,
		LayoutMode: "center_fit", KeepSubjectComplete: true, OutputFormat: template.OutputFormat, Margin: template.Margin,
	})
	if err != nil {
		return err
	}
	lineColor, err := parseBackground(template.AnnotationColor)
	if err != nil {
		return err
	}
	w, h := template.CanvasWidth, template.CanvasHeight
	drawDimension(canvas, image.Pt(w*19/100, h*895/1000), image.Pt(w*81/100, h*895/1000), formatValue(size.Width, size.Unit), lineColor)
	drawDimension(canvas, image.Pt(w*105/1000, h/5), image.Pt(w*105/1000, h*4/5), formatValue(size.Height, size.Unit), lineColor)
	if size.Depth > 0 {
		drawDimension(canvas, image.Pt(w*735/1000, h*215/1000), image.Pt(w*89/100, h*105/1000), formatValue(size.Depth, size.Unit), lineColor)
	}
	return encode(outputPath, canvas, template.OutputFormat)
}

// NormalizeSizeTemplateConfig 解析尺寸图模板并补齐兼容默认值。
func NormalizeSizeTemplateConfig(data []byte) (SizeTemplateConfig, error) {
	config := SizeTemplateConfig{
		CanvasWidth: 1000, CanvasHeight: 1000, Background: "#FFFFFF", OutputFormat: "jpeg",
		Margin: 150, AnnotationColor: "#2563EB",
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return SizeTemplateConfig{}, err
	}
	if config.CanvasWidth < 64 || config.CanvasWidth > 4096 || config.CanvasHeight < 64 || config.CanvasHeight > 4096 {
		return SizeTemplateConfig{}, fmt.Errorf("尺寸图画布宽高必须在 64 到 4096 像素之间")
	}
	if config.Margin < 0 || config.Margin*2 >= config.CanvasWidth || config.Margin*2 >= config.CanvasHeight {
		return SizeTemplateConfig{}, fmt.Errorf("尺寸图边距不能超过画布范围")
	}
	if config.OutputFormat != "jpeg" && config.OutputFormat != "png" && config.OutputFormat != "webp" {
		return SizeTemplateConfig{}, fmt.Errorf("尺寸图输出格式不受支持")
	}
	if config.Background == "transparent" && config.OutputFormat == "jpeg" {
		return SizeTemplateConfig{}, fmt.Errorf("JPEG 不支持透明背景")
	}
	if _, err := parseBackground(config.Background); err != nil {
		return SizeTemplateConfig{}, err
	}
	if _, err := parseBackground(config.AnnotationColor); err != nil || config.AnnotationColor == "transparent" {
		return SizeTemplateConfig{}, fmt.Errorf("尺寸标注颜色必须是 #RRGGBB")
	}
	return config, nil
}

func decode(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开原图失败: %w", err)
	}
	defer file.Close()
	value, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("解码原图失败: %w", err)
	}
	return value, nil
}

func compose(source image.Image, config ProcessConfig) (*image.NRGBA, error) {
	background, err := parseBackground(config.Background)
	if err != nil {
		return nil, err
	}
	canvas := image.NewNRGBA(image.Rect(0, 0, config.CanvasWidth, config.CanvasHeight))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: background}, image.Point{}, draw.Src)
	availableWidth := config.CanvasWidth - config.Margin*2
	availableHeight := config.CanvasHeight - config.Margin*2
	if availableWidth <= 0 || availableHeight <= 0 {
		return nil, fmt.Errorf("边距不能超过画布范围")
	}
	sourceBounds := source.Bounds()
	scale := math.Min(float64(availableWidth)/float64(sourceBounds.Dx()), float64(availableHeight)/float64(sourceBounds.Dy()))
	width := max(1, int(math.Round(float64(sourceBounds.Dx())*scale)))
	height := max(1, int(math.Round(float64(sourceBounds.Dy())*scale)))
	x := (config.CanvasWidth - width) / 2
	y := (config.CanvasHeight - height) / 2
	xdraw.CatmullRom.Scale(canvas, image.Rect(x, y, x+width, y+height), source, sourceBounds, draw.Over, nil)
	return canvas, nil
}

func encode(path string, value image.Image, format string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer file.Close()
	switch format {
	case "jpeg", "jpg":
		err = jpeg.Encode(file, value, &jpeg.Options{Quality: 90})
	case "png":
		err = png.Encode(file, value)
	case "webp":
		err = nativewebp.Encode(file, value, nil)
	default:
		return fmt.Errorf("不支持的输出格式 %q", format)
	}
	if err != nil {
		return fmt.Errorf("编码输出图片失败: %w", err)
	}
	return nil
}

func parseBackground(value string) (color.NRGBA, error) {
	if value == "transparent" {
		return color.NRGBA{}, nil
	}
	hex := strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(hex) != 6 {
		return color.NRGBA{}, fmt.Errorf("背景色必须是 transparent 或 #RRGGBB")
	}
	parsed, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("背景色格式无效")
	}
	return color.NRGBA{R: uint8(parsed >> 16), G: uint8(parsed >> 8), B: uint8(parsed), A: 255}, nil
}

func drawDimension(canvas *image.NRGBA, start, end image.Point, label string, lineColor color.NRGBA) {
	drawLine(canvas, start, end, lineColor, 4)
	dx, dy := float64(end.X-start.X), float64(end.Y-start.Y)
	length := math.Hypot(dx, dy)
	if length == 0 {
		return
	}
	ux, uy := dx/length, dy/length
	for _, point := range []image.Point{start, end} {
		direction := 1.0
		if point == end {
			direction = -1
		}
		for _, angle := range []float64{-0.55, 0.55} {
			cosA, sinA := math.Cos(angle), math.Sin(angle)
			vx := (ux*cosA - uy*sinA) * direction
			vy := (ux*sinA + uy*cosA) * direction
			tip := image.Pt(point.X+int(vx*22), point.Y+int(vy*22))
			drawLine(canvas, point, tip, lineColor, 4)
		}
	}
	drawLabel(canvas, image.Pt((start.X+end.X)/2, (start.Y+end.Y)/2), label, lineColor)
}

func drawLine(canvas *image.NRGBA, start, end image.Point, lineColor color.NRGBA, thickness int) {
	dx, dy := end.X-start.X, end.Y-start.Y
	steps := max(abs(dx), abs(dy))
	if steps == 0 {
		return
	}
	for i := 0; i <= steps; i++ {
		x := start.X + dx*i/steps
		y := start.Y + dy*i/steps
		draw.Draw(canvas, image.Rect(x-thickness/2, y-thickness/2, x+thickness/2+1, y+thickness/2+1), &image.Uniform{C: lineColor}, image.Point{}, draw.Src)
	}
}

func drawLabel(canvas *image.NRGBA, center image.Point, label string, textColor color.NRGBA) {
	fontValue, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return
	}
	face, err := opentype.NewFace(fontValue, &opentype.FaceOptions{Size: 28, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return
	}
	defer face.Close()
	width := font.MeasureString(face, label).Ceil()
	rect := image.Rect(center.X-width/2-14, center.Y-22, center.X+width/2+14, center.Y+20)
	draw.Draw(canvas, rect, &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	drawer := &font.Drawer{Dst: canvas, Src: &image.Uniform{C: textColor}, Face: face, Dot: fixed.P(rect.Min.X+14, center.Y+10)}
	drawer.DrawString(label)
}

func formatValue(value float64, unit string) string {
	return strconv.FormatFloat(value, 'f', -1, 64) + " " + unit
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
