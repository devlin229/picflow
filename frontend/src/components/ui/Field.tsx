import { useEffect, useRef } from "react";
import type { InputHTMLAttributes, ReactNode, SelectHTMLAttributes } from "react";

type InputFieldProps = InputHTMLAttributes<HTMLInputElement> & { label: string };

export function InputField({ label, className = "", ...props }: InputFieldProps) {
  return (
    <label className={`field ${className}`}>
      <span>{label}</span>
      <input {...props} />
    </label>
  );
}

type SelectFieldProps = SelectHTMLAttributes<HTMLSelectElement> & { label: string };

export function SelectField({ label, className = "", children, ...props }: SelectFieldProps) {
  return (
    <label className={`field ${className}`}>
      <span>{label}</span>
      <select {...props}>{children}</select>
    </label>
  );
}

type RangeFieldProps = Omit<InputHTMLAttributes<HTMLInputElement>, "type" | "onChange"> & {
  label: ReactNode;
  help?: ReactNode;
  onValueChange: (value: number) => void;
};

function numericRangeValue(value: string | number | readonly string[] | undefined, fallback: number) {
  const parsed = typeof value === "number" ? value : Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

export function RangeField({ label, help, className = "", min = 0, max = 100, step = 1, value, onValueChange, ...props }: RangeFieldProps) {
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const input = inputRef.current;
    if (!input) return undefined;
    const handleWheel = (event: WheelEvent) => {
      if (event.deltaY === 0) return;
      const minimum = numericRangeValue(min, 0);
      const maximum = numericRangeValue(max, 100);
      const increment = step === "any" ? 1 : Math.max(Number.EPSILON, numericRangeValue(step, 1));
      const current = Number.isFinite(input.valueAsNumber) ? input.valueAsNumber : numericRangeValue(value, minimum);
      const direction = event.deltaY < 0 ? 1 : -1;
      const next = Math.min(maximum, Math.max(minimum, current + direction * increment));
      event.preventDefault();
      if (next !== current) onValueChange(next);
    };
    input.addEventListener("wheel", handleWheel, { passive: false });
    return () => input.removeEventListener("wheel", handleWheel);
  }, [max, min, onValueChange, step, value]);

  return (
    <label className={`field range-field ${className}`}>
      <span>{label}</span>
      <input
        {...props}
        type="range"
        min={min}
        max={max}
        step={step}
        value={value}
        ref={inputRef}
        title="悬停后滚动鼠标滚轮，或拖动滑块调整"
        onChange={(event) => onValueChange(event.currentTarget.valueAsNumber)}
      />
      {help && <small>{help}</small>}
    </label>
  );
}
