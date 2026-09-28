import type { InputHTMLAttributes, SelectHTMLAttributes } from "react";

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
