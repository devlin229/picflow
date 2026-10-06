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
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/HugoSmits86/nativewebp"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/webp"
)

// ProcessConfig 描述规则型图片处理参数。
type ProcessConfig struct {
	CanvasWidth         int                  `json:"canvas_width"`
	CanvasHeight        int                  `json:"canvas_height"`
	Background          string               `json:"background"`
	LayoutMode          string               `json:"layout_mode"`
	KeepSubjectComplete bool                 `json:"keep_subject_complete"`
	OutputFormat        string               `json:"output_format"`
	MarginMode          string               `json:"margin_mode"`
	Margin              int                  `json:"margin"`
	OutputQuality       int                  `json:"output_quality"`
	AIBackground        bool                 `json:"ai_background"`
	AIBackgroundPrompt  string               `json:"ai_background_prompt"`
	Specification       *SpecificationConfig `json:"specification,omitempty"`
}

// Specification 描述规格表格中的一行参数。
type Specification struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// SpecificationConfig 描述规格表格内容和位置。
type SpecificationConfig struct {
	Items       []Specification    `json:"items"`
	TablePreset string             `json:"table_preset"`
	TableX      float64            `json:"table_x"`
	TableY      float64            `json:"table_y"`
	TableWidth  float64            `json:"table_width"`
	Style       SpecificationStyle `json:"style"`
}

// SpecificationStyle 描述规格表格的可自定义视觉样式。
type SpecificationStyle struct {
	Style           string `json:"style"`
	BorderWidth     int    `json:"border_width"`
	BorderColor     string `json:"border_color"`
	BackgroundColor string `json:"background_color"`
	TextColor       string `json:"text_color"`
}

// SizeTemplateConfig 描述规格图模板对画布和表格样式的控制。
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
	if config.Specification != nil {
		borderColor, parseErr := parseBackground(config.Specification.Style.BorderColor)
		if parseErr != nil {
			return fmt.Errorf("规格表格边框色无效: %w", parseErr)
		}
		backgroundColor, parseErr := parseBackground(config.Specification.Style.BackgroundColor)
		if parseErr != nil {
			return fmt.Errorf("规格表格背景色无效: %w", parseErr)
		}
		textColor, parseErr := parseBackground(config.Specification.Style.TextColor)
		if parseErr != nil {
			return fmt.Errorf("规格表格文字色无效: %w", parseErr)
		}
		if err = drawSpecificationTable(canvas, *config.Specification, borderColor, backgroundColor, textColor); err != nil {
			return err
		}
	}
	return encode(outputPath, canvas, config.OutputFormat, config.OutputQuality)
}

// SpecificationChart 生成带可拖动规格表格的商品图。
func SpecificationChart(sourcePath, outputPath string, specification SpecificationConfig, template SizeTemplateConfig) error {
	if len(specification.Items) == 0 || len(specification.Items) > 10 {
		return fmt.Errorf("规格参数数量必须在 1 到 10 项之间")
	}
	if specification.TableX < 0 || specification.TableX > 1 || specification.TableY < 0 || specification.TableY > 1 {
		return fmt.Errorf("规格表格位置必须在图片范围内")
	}
	if specification.TableWidth < 0.24 || specification.TableWidth > 0.8 {
		return fmt.Errorf("规格表格宽度必须在画布宽度的 24%% 到 80%% 之间")
	}
	source, err := decode(sourcePath)
	if err != nil {
		return err
	}
	canvas, err := compose(source, ProcessConfig{
		CanvasWidth: template.CanvasWidth, CanvasHeight: template.CanvasHeight, Background: template.Background,
		LayoutMode: "contain", KeepSubjectComplete: true, OutputFormat: template.OutputFormat, Margin: template.Margin,
	})
	if err != nil {
		return err
	}
	borderColor, err := parseBackground(specification.Style.BorderColor)
	if err != nil {
		return fmt.Errorf("规格表格边框色无效: %w", err)
	}
	backgroundColor, err := parseBackground(specification.Style.BackgroundColor)
	if err != nil {
		return fmt.Errorf("规格表格背景色无效: %w", err)
	}
	textColor, err := parseBackground(specification.Style.TextColor)
	if err != nil {
		return fmt.Errorf("规格表格文字色无效: %w", err)
	}
	if err := drawSpecificationTable(canvas, specification, borderColor, backgroundColor, textColor); err != nil {
		return err
	}
	return encode(outputPath, canvas, template.OutputFormat, 90)
}

// NormalizeSizeTemplateConfig 解析规格图模板并补齐兼容默认值。
func NormalizeSizeTemplateConfig(data []byte) (SizeTemplateConfig, error) {
	config := SizeTemplateConfig{
		CanvasWidth: 1000, CanvasHeight: 1000, Background: "#FFFFFF", OutputFormat: "jpeg",
		Margin: 80, AnnotationColor: "#2563EB",
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return SizeTemplateConfig{}, err
	}
	if config.CanvasWidth < 64 || config.CanvasWidth > 4096 || config.CanvasHeight < 64 || config.CanvasHeight > 4096 {
		return SizeTemplateConfig{}, fmt.Errorf("规格图画布宽高必须在 64 到 4096 像素之间")
	}
	if config.Margin < 0 || config.Margin*2 >= config.CanvasWidth || config.Margin*2 >= config.CanvasHeight {
		return SizeTemplateConfig{}, fmt.Errorf("规格图边距不能超过画布范围")
	}
	if config.OutputFormat != "jpeg" && config.OutputFormat != "png" && config.OutputFormat != "webp" {
		return SizeTemplateConfig{}, fmt.Errorf("规格图输出格式不受支持")
	}
	if config.Background == "transparent" && config.OutputFormat == "jpeg" {
		return SizeTemplateConfig{}, fmt.Errorf("JPEG 不支持透明背景")
	}
	if _, err := parseBackground(config.Background); err != nil {
		return SizeTemplateConfig{}, err
	}
	if _, err := parseBackground(config.AnnotationColor); err != nil || config.AnnotationColor == "transparent" {
		return SizeTemplateConfig{}, fmt.Errorf("规格表格颜色必须是 #RRGGBB")
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
	margin := config.Margin
	if config.LayoutMode != "contain" && config.LayoutMode != "center_fit" {
		margin = 0
	}
	availableWidth := config.CanvasWidth - margin*2
	availableHeight := config.CanvasHeight - margin*2
	if availableWidth <= 0 || availableHeight <= 0 {
		return nil, fmt.Errorf("边距不能超过画布范围")
	}
	sourceBounds := source.Bounds()
	scale := math.Min(float64(availableWidth)/float64(sourceBounds.Dx()), float64(availableHeight)/float64(sourceBounds.Dy()))
	if config.LayoutMode != "contain" && config.LayoutMode != "center_fit" {
		scale = math.Max(float64(availableWidth)/float64(sourceBounds.Dx()), float64(availableHeight)/float64(sourceBounds.Dy()))
	}
	width := max(1, int(math.Round(float64(sourceBounds.Dx())*scale)))
	height := max(1, int(math.Round(float64(sourceBounds.Dy())*scale)))
	x := (config.CanvasWidth - width) / 2
	y := (config.CanvasHeight - height) / 2
	if config.LayoutMode == "cover-top" {
		y = 0
	}
	if config.LayoutMode == "cover-bottom" {
		y = config.CanvasHeight - height
	}
	xdraw.CatmullRom.Scale(canvas, image.Rect(x, y, x+width, y+height), source, sourceBounds, draw.Over, nil)
	return canvas, nil
}

func encode(path string, value image.Image, format string, quality int) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer file.Close()
	switch format {
	case "jpeg", "jpg":
		if quality == 0 {
			quality = 90
		}
		err = jpeg.Encode(file, value, &jpeg.Options{Quality: quality})
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

func drawSpecificationTable(canvas *image.NRGBA, specification SpecificationConfig, borderColor, backgroundColor, textColor color.NRGBA) error {
	fontValue, err := loadSpecificationFont()
	if err != nil {
		return err
	}
	shortSide := min(canvas.Bounds().Dx(), canvas.Bounds().Dy())
	tableScale := math.Max(0.78, math.Min(1.45, specification.TableWidth/0.42))
	fontSize := math.Max(14, math.Min(40, float64(shortSide)/36*tableScale))
	face, err := opentype.NewFace(fontValue, &opentype.FaceOptions{Size: fontSize, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return fmt.Errorf("创建规格表格字体失败: %w", err)
	}
	defer face.Close()

	padding := max(10, int(math.Round(fontSize*0.62)))
	rowHeight := max(34, int(math.Round(fontSize+float64(padding))))
	tableWidth := min(max(180, int(math.Round(float64(canvas.Bounds().Dx())*specification.TableWidth))), canvas.Bounds().Dx()-24)
	tableHeight := rowHeight * len(specification.Items)
	if tableWidth < 180 || tableHeight > canvas.Bounds().Dy()-24 {
		return fmt.Errorf("规格图画布过小，无法绘制规格表格")
	}

	centerX := int(math.Round(specification.TableX * float64(canvas.Bounds().Dx())))
	centerY := int(math.Round(specification.TableY * float64(canvas.Bounds().Dy())))
	left := min(max(12, centerX-tableWidth/2), canvas.Bounds().Dx()-tableWidth-12)
	top := min(max(12, centerY-tableHeight/2), canvas.Bounds().Dy()-tableHeight-12)
	tableRect := image.Rect(left, top, left+tableWidth, top+tableHeight)
	if backgroundColor.A > 0 {
		draw.Draw(canvas, tableRect, &image.Uniform{C: backgroundColor}, image.Point{}, draw.Over)
	}
	borderWidth := min(8, max(1, specification.Style.BorderWidth))
	drawBorder(canvas, tableRect, borderColor, borderWidth)

	divider := borderColor
	divider.A = 96
	dividerWidth := max(1, borderWidth/2)
	labelWidth := tableWidth * 38 / 100
	for index, item := range specification.Items {
		rowTop := top + index*rowHeight
		if index > 0 {
			draw.Draw(canvas, image.Rect(left+borderWidth, rowTop, left+tableWidth-borderWidth, rowTop+dividerWidth), &image.Uniform{C: divider}, image.Point{}, draw.Over)
		}
		baseline := rowTop + (rowHeight+face.Metrics().Ascent.Ceil()-face.Metrics().Descent.Ceil())/2
		label := fitText(face, item.Label, labelWidth-padding*2)
		value := fitText(face, item.Value, tableWidth-labelWidth-padding*2)
		draw.Draw(canvas, image.Rect(left+labelWidth, rowTop+dividerWidth, left+labelWidth+dividerWidth, rowTop+rowHeight), &image.Uniform{C: divider}, image.Point{}, draw.Over)
		drawText(canvas, face, label, left+padding, baseline, textColor)
		drawText(canvas, face, value, left+labelWidth+padding, baseline, textColor)
	}
	return nil
}

func drawBorder(canvas *image.NRGBA, rect image.Rectangle, borderColor color.NRGBA, thickness int) {
	draw.Draw(canvas, image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+thickness), &image.Uniform{C: borderColor}, image.Point{}, draw.Src)
	draw.Draw(canvas, image.Rect(rect.Min.X, rect.Max.Y-thickness, rect.Max.X, rect.Max.Y), &image.Uniform{C: borderColor}, image.Point{}, draw.Src)
	draw.Draw(canvas, image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+thickness, rect.Max.Y), &image.Uniform{C: borderColor}, image.Point{}, draw.Src)
	draw.Draw(canvas, image.Rect(rect.Max.X-thickness, rect.Min.Y, rect.Max.X, rect.Max.Y), &image.Uniform{C: borderColor}, image.Point{}, draw.Src)
}

func drawText(canvas *image.NRGBA, face font.Face, value string, x, baseline int, textColor color.NRGBA) {
	drawer := &font.Drawer{Dst: canvas, Src: &image.Uniform{C: textColor}, Face: face, Dot: fixed.P(x, baseline)}
	drawer.DrawString(value)
}

func fitText(face font.Face, value string, maxWidth int) string {
	if font.MeasureString(face, value).Ceil() <= maxWidth {
		return value
	}
	ellipsis := "…"
	runes := []rune(value)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		candidate := string(runes) + ellipsis
		if font.MeasureString(face, candidate).Ceil() <= maxWidth {
			return candidate
		}
	}
	return ellipsis
}

var (
	specificationFontOnce  sync.Once
	specificationFontValue *opentype.Font
	specificationFontErr   error
)

func loadSpecificationFont() (*opentype.Font, error) {
	specificationFontOnce.Do(func() {
		specificationFontValue, specificationFontErr = findSpecificationFont()
	})
	return specificationFontValue, specificationFontErr
}

func findSpecificationFont() (*opentype.Font, error) {
	candidates := []string{
		strings.TrimSpace(os.Getenv("PICFLOW_FONT_PATH")),
		"/usr/share/fonts/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/System/Library/Fonts/STHeiti Medium.ttc",
		"/System/Library/Fonts/Hiragino Sans GB.ttc",
	}
	if matches, _ := filepath.Glob("/usr/share/fonts/**/*CJK*Regular*.ttc"); len(matches) > 0 {
		candidates = append(candidates, matches...)
	}
	for _, path := range candidates {
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if strings.EqualFold(filepath.Ext(path), ".ttc") {
			collection, parseErr := opentype.ParseCollection(data)
			if parseErr != nil || collection.NumFonts() == 0 {
				continue
			}
			value, fontErr := collection.Font(0)
			if fontErr == nil {
				return value, nil
			}
			continue
		}
		value, parseErr := opentype.Parse(data)
		if parseErr == nil {
			return value, nil
		}
	}
	return nil, fmt.Errorf("未找到支持中文的字体，请安装 Noto Sans CJK 或设置 PICFLOW_FONT_PATH")
}
