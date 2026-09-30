import { useEffect, useRef, useState } from "react";
import type { CSSProperties, PointerEvent as ReactPointerEvent } from "react";
import type { ImageAsset, ProcessConfig, SpecificationTablePosition, TaskSpecification } from "../types";

type ProcessingPreviewProps = {
  asset: ImageAsset;
  config: ProcessConfig;
  specification?: TaskSpecification;
  onSpecificationPositionChange?: (position: SpecificationTablePosition) => void;
};

type ResizeEdge = "top" | "right" | "bottom" | "left" | "top-left" | "top-right" | "bottom-left" | "bottom-right";

const resizeEdges: ResizeEdge[] = ["top", "right", "bottom", "left", "top-left", "top-right", "bottom-left", "bottom-right"];

function parseColor(value: string) {
  if (value === "transparent") return [0, 0, 0, 0] as const;
  const hex = value.replace("#", "");
  return [Number.parseInt(hex.slice(0, 2), 16), Number.parseInt(hex.slice(2, 4), 16), Number.parseInt(hex.slice(4, 6), 16), 255] as const;
}

function removeConnectedBackground(context: CanvasRenderingContext2D, width: number, height: number, tolerance: number) {
  const image = context.getImageData(0, 0, width, height);
  const data = image.data;
  const corners = [[0, 0], [width - 1, 0], [0, height - 1], [width - 1, height - 1]] as const;
  const colors = corners.map(([x, y]) => {
    const offset = (y * width + x) * 4;
    return [data[offset], data[offset + 1], data[offset + 2]] as const;
  });
  const channelTolerance = Math.round(tolerance / 100 * 255);
  const score = colors.map((candidate) => colors.filter((color) => color.every((channel, index) => Math.abs(channel - candidate[index]) <= channelTolerance)).length);
  const reference = colors[score.indexOf(Math.max(...score))];
  const visited = new Uint8Array(width * height);
  const queue: number[] = [];
  const enqueue = (x: number, y: number) => {
    const index = y * width + x;
    if (visited[index]) return;
    visited[index] = 1;
    const offset = index * 4;
    const matches = data[offset + 3] < 250 || reference.every((channel, colorIndex) => Math.abs(data[offset + colorIndex] - channel) <= channelTolerance);
    if (matches) queue.push(index);
  };
  for (let x = 0; x < width; x += 1) { enqueue(x, 0); enqueue(x, height - 1); }
  for (let y = 1; y < height - 1; y += 1) { enqueue(0, y); enqueue(width - 1, y); }
  for (let cursor = 0; cursor < queue.length; cursor += 1) {
    const index = queue[cursor];
    data[index * 4 + 3] = 0;
    const x = index % width;
    const y = Math.floor(index / width);
    if (x > 0) enqueue(x - 1, y);
    if (x + 1 < width) enqueue(x + 1, y);
    if (y > 0) enqueue(x, y - 1);
    if (y + 1 < height) enqueue(x, y + 1);
  }
  context.putImageData(image, 0, 0);
}

export function ProcessingPreview({ asset, config, specification, onSpecificationPositionChange }: ProcessingPreviewProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const previewRef = useRef<HTMLDivElement>(null);
  const tableRef = useRef<HTMLDivElement>(null);
  const resizeSessionRef = useRef<{
    edge: ResizeEdge;
    startX: number;
    startY: number;
    startWidth: number;
    tableWidth: number;
    tableHeight: number;
  } | null>(null);
  const [dragging, setDragging] = useState(false);
  const [resizing, setResizing] = useState<ResizeEdge | null>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const context = canvas.getContext("2d", { willReadFrequently: config.replaceSimpleBackground });
    if (!context) return;
    const image = new Image();
    image.onload = () => {
      const previewWidth = Math.min(720, config.width);
      const previewHeight = Math.max(1, Math.round(previewWidth * config.height / config.width));
      canvas.width = previewWidth;
      canvas.height = previewHeight;
      context.clearRect(0, 0, previewWidth, previewHeight);
      const [red, green, blue, alpha] = parseColor(config.background);
      context.fillStyle = `rgba(${red}, ${green}, ${blue}, ${alpha / 255})`;
      context.fillRect(0, 0, previewWidth, previewHeight);

      const margin = config.layoutMode === "contain"
        ? (config.marginMode === "auto" ? Math.round(Math.min(previewWidth, previewHeight) * 0.08) : config.margin * previewWidth / config.width)
        : 0;
      const availableWidth = previewWidth - margin * 2;
      const availableHeight = previewHeight - margin * 2;
      const cover = config.layoutMode !== "contain";
      const scale = cover
        ? Math.max(availableWidth / image.naturalWidth, availableHeight / image.naturalHeight)
        : Math.min(availableWidth / image.naturalWidth, availableHeight / image.naturalHeight);
      const drawWidth = image.naturalWidth * scale;
      const drawHeight = image.naturalHeight * scale;
      const x = (previewWidth - drawWidth) / 2;
      let y = (previewHeight - drawHeight) / 2;
      if (config.layoutMode === "cover-top") y = 0;
      if (config.layoutMode === "cover-bottom") y = previewHeight - drawHeight;

      if (!config.replaceSimpleBackground) {
        context.drawImage(image, x, y, drawWidth, drawHeight);
        return;
      }
      const sourceCanvas = document.createElement("canvas");
      const sampleScale = Math.min(1, 1600 / Math.max(image.naturalWidth, image.naturalHeight));
      sourceCanvas.width = Math.max(1, Math.round(image.naturalWidth * sampleScale));
      sourceCanvas.height = Math.max(1, Math.round(image.naturalHeight * sampleScale));
      const sourceContext = sourceCanvas.getContext("2d", { willReadFrequently: true });
      if (!sourceContext) return;
      sourceContext.drawImage(image, 0, 0, sourceCanvas.width, sourceCanvas.height);
      removeConnectedBackground(sourceContext, sourceCanvas.width, sourceCanvas.height, config.backgroundTolerance);
      context.drawImage(sourceCanvas, x, y, drawWidth, drawHeight);
    };
    image.src = asset.url;
    return () => { image.onload = null; };
  }, [asset, config]);

  const changePosition = (position: SpecificationTablePosition) => onSpecificationPositionChange?.(position);

  const clampPosition = (x: number, y: number) => {
    const preview = previewRef.current?.getBoundingClientRect();
    const table = tableRef.current?.getBoundingClientRect();
    if (!preview || !table) return { x: Math.min(0.9, Math.max(0.1, x)), y: Math.min(0.9, Math.max(0.1, y)) };
    const minimumX = Math.min(0.5, (table.width / 2 + 12) / preview.width);
    const minimumY = Math.min(0.5, (table.height / 2 + 12) / preview.height);
    return {
      x: Math.min(1 - minimumX, Math.max(minimumX, x)),
      y: Math.min(1 - minimumY, Math.max(minimumY, y)),
    };
  };

  const moveTable = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (!specification) return;
    const preview = previewRef.current?.getBoundingClientRect();
    if (!preview) return;
    const next = clampPosition((event.clientX - preview.left) / preview.width, (event.clientY - preview.top) / preview.height);
    changePosition({ ...specification.tablePosition, preset: "custom", ...next });
  };

  const startDragging = (event: ReactPointerEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
    setDragging(true);
    moveTable(event);
  };

  const resizeTable = (event: ReactPointerEvent<HTMLDivElement>) => {
    const session = resizeSessionRef.current;
    if (!specification || !session) return;
    const horizontalDirection = session.edge.includes("left") ? -1 : session.edge.includes("right") ? 1 : 0;
    const verticalDirection = session.edge.includes("top") ? -1 : session.edge.includes("bottom") ? 1 : 0;
    const horizontalScale = horizontalDirection
      ? (session.tableWidth + horizontalDirection * (event.clientX - session.startX) * 2) / session.tableWidth
      : 1;
    const verticalScale = verticalDirection
      ? (session.tableHeight + verticalDirection * (event.clientY - session.startY) * 2) / session.tableHeight
      : 1;
    const scale = horizontalDirection && verticalDirection
      ? (Math.abs(horizontalScale - 1) >= Math.abs(verticalScale - 1) ? horizontalScale : verticalScale)
      : horizontalDirection ? horizontalScale : verticalScale;
    const availableWidth = 2 * Math.min(specification.tablePosition.x - 0.02, 0.98 - specification.tablePosition.x);
    const width = Math.min(0.8, Math.max(0.24, Math.min(availableWidth, session.startWidth * scale)));
    changePosition({ ...specification.tablePosition, preset: "custom", width });
  };

  const startResizing = (event: ReactPointerEvent<HTMLDivElement>, edge: ResizeEdge) => {
    if (!specification) return;
    const table = tableRef.current?.getBoundingClientRect();
    if (!table) return;
    event.preventDefault();
    event.stopPropagation();
    event.currentTarget.setPointerCapture(event.pointerId);
    resizeSessionRef.current = {
      edge,
      startX: event.clientX,
      startY: event.clientY,
      startWidth: specification.tablePosition.width,
      tableWidth: table.width,
      tableHeight: table.height,
    };
    setResizing(edge);
  };

  const stopResizing = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
    resizeSessionRef.current = null;
    setResizing(null);
  };

  return (
    <div
      className="processing-preview__canvas-wrap"
      ref={previewRef}
      style={{ aspectRatio: `${config.width} / ${config.height}`, width: "auto", height: "100%", maxWidth: `${Math.min(720, 520 * config.width / config.height)}px`, maxHeight: "100%" }}
    >
      <canvas ref={canvasRef} className="processing-preview__canvas" aria-label={`${asset.name} 处理预览`} />
      {specification && <div
        className={`spec-overlay-table ${dragging || resizing ? "spec-overlay-table--dragging" : ""}`}
        ref={tableRef}
        style={{
          left: `${specification.tablePosition.x * 100}%`,
          top: `${specification.tablePosition.y * 100}%`,
          width: `${specification.tablePosition.width * 100}%`,
          color: specification.tableStyle.textColor,
          backgroundColor: specification.tableStyle.backgroundColor,
          borderColor: specification.tableStyle.borderColor,
          borderWidth: `${specification.tableStyle.borderWidth}px`,
          "--spec-scale": Math.min(1.45, Math.max(0.78, specification.tablePosition.width / 0.42)),
          "--spec-border": specification.tableStyle.borderColor,
          "--spec-text": specification.tableStyle.textColor,
        } as CSSProperties}
        onPointerDown={startDragging}
        onPointerMove={(event) => dragging && moveTable(event)}
        onPointerUp={(event) => {
          if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
          setDragging(false);
        }}
        onPointerCancel={() => setDragging(false)}
      >
        <table><tbody>
          {specification.specifications.map((item) => <tr key={item.id}><th>{item.label}</th><td>{item.value}</td></tr>)}
        </tbody></table>
        {resizeEdges.map((edge) => <div
          aria-hidden="true"
          className={`spec-overlay-table__resize spec-overlay-table__resize--${edge}`}
          key={edge}
          onPointerDown={(event) => startResizing(event, edge)}
          onPointerMove={(event) => resizing === edge && resizeTable(event)}
          onPointerUp={stopResizing}
          onPointerCancel={(event) => stopResizing(event)}
        />)}
      </div>}
    </div>
  );
}
