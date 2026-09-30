import { X } from "lucide-react";

type ImageCardProps = {
  src: string;
  name: string;
  meta: string;
  selected?: boolean;
  onClick?: () => void;
  onRemove?: () => void;
};

export function ImageCard({ src, name, meta, selected, onClick, onRemove }: ImageCardProps) {
  return (
    <article className={`image-card ${onClick ? "image-card--selectable" : ""} ${selected ? "image-card--selected" : ""}`}>
      {onClick && <button type="button" className="image-card__select" aria-label={`预览 ${name}`} aria-pressed={selected} title={`预览 ${name}`} onClick={onClick} />}
      {onRemove && <button type="button" className="image-card__remove" aria-label={`删除 ${name}`} onClick={onRemove}><X size={14} /></button>}
      <span className="image-card__preview"><img src={src} alt={name} /></span>
      <strong title={name}>{name}</strong>
      <small>{meta}</small>
    </article>
  );
}
