package imageproc

import (
	"fmt"
	"regexp"
	"strings"
)

var backgroundColorInstruction = regexp.MustCompile(`(?:换成|换为|替换为|改成|改为|变成|设置为)\s*(黑色|白色|红色|绿色|蓝色|黄色|#?[0-9a-fA-F]{6})(?:背景|底色)?[。！!\s]*$`)

// ResolveAIBackground 将明确换色指令统一为画布背景色，保留其他场景描述。
func ResolveAIBackground(background, prompt string) (string, string) {
	prompt = strings.TrimSpace(prompt)
	match := backgroundColorInstruction.FindStringSubmatch(prompt)
	if match == nil {
		return background, prompt
	}
	colors := map[string]string{"黑色": "#000000", "白色": "#FFFFFF", "红色": "#FF0000", "绿色": "#00FF00", "蓝色": "#0000FF", "黄色": "#FFFF00"}
	if color, ok := colors[match[1]]; ok {
		return color, ""
	}
	return "#" + strings.ToUpper(strings.TrimPrefix(match[1], "#")), ""
}

// ValidateAIBackground 对纯色背景进行边缘采样，避免明显未换色的结果被标记成功。
// 这是背景颜色的初步校验，不承担商品语义或商品细节验收。
func ValidateAIBackground(path, background string) error {
	source, err := decode(path)
	if err != nil {
		return fmt.Errorf("AI 返回图片无法解码")
	}
	target, err := parseBackground(background)
	if err != nil {
		return err
	}
	bounds := source.Bounds()
	matched, total := 0, 0
	for side := 0; side < 4; side++ {
		for index := 0; index < 100; index++ {
			x, y := bounds.Min.X+index*(bounds.Dx()-1)/99, bounds.Min.Y
			switch side {
			case 1:
				y = bounds.Max.Y - 1
			case 2:
				x, y = bounds.Min.X, bounds.Min.Y+index*(bounds.Dy()-1)/99
			case 3:
				x, y = bounds.Max.X-1, bounds.Min.Y+index*(bounds.Dy()-1)/99
			}
			r, g, b, a := source.At(x, y).RGBA()
			// 将透明像素按最终目标画布颜色合成后再检查。
			alpha := int(a >> 8)
			rgb := []int{int(r>>8) + int(target.R)*(255-alpha)/255, int(g>>8) + int(target.G)*(255-alpha)/255, int(b>>8) + int(target.B)*(255-alpha)/255}
			if absInt(rgb[0]-int(target.R)) <= 40 && absInt(rgb[1]-int(target.G)) <= 40 && absInt(rgb[2]-int(target.B)) <= 40 {
				matched++
			}
			total++
		}
	}
	if matched*100 < total*60 {
		return fmt.Errorf("AI 返回图片的背景未达到目标颜色 %s，已保留模型原图；请新建任务调整指令，不会自动再次扣费", background)
	}
	return nil
}
