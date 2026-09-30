// 仅识别明确的单一换色指令，场景描述交由图片模型处理。
export function backgroundColorFromInstruction(prompt: string): string | undefined {
  const match = prompt.trim().match(/(?:换成|换为|替换为|改成|改为|变成|设置为)\s*(黑色|白色|红色|绿色|蓝色|黄色|#?[0-9a-fA-F]{6})(?:背景|底色)?[。！!\s]*$/);
  if (!match) return undefined;
  const colors: Record<string, string> = { 黑色: "#000000", 白色: "#FFFFFF", 红色: "#FF0000", 绿色: "#00FF00", 蓝色: "#0000FF", 黄色: "#FFFF00" };
  return colors[match[1]] ?? `#${match[1].replace("#", "").toUpperCase()}`;
}
