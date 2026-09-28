type ImageCardProps = {
  src: string;
  name: string;
  meta: string;
  selected?: boolean;
  onClick?: () => void;
};

export function ImageCard({ src, name, meta, selected, onClick }: ImageCardProps) {
  return (
    <button type="button" className={`image-card ${selected ? "image-card--selected" : ""}`} onClick={onClick}>
      <span className="image-card__preview"><img src={src} alt={name} /></span>
      <strong title={name}>{name}</strong>
      <small>{meta}</small>
    </button>
  );
}
