export type ApiResponse<T> = {
  code: number;
  msg: string;
  data: T;
};

export class ApiError extends Error {
  readonly code: number;
  readonly status: number;

  constructor(message: string, code: number, status: number) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
  }
}

const apiBaseURL = (import.meta.env.VITE_API_BASE_URL ?? "").replace(/\/$/, "");

export function apiURL(path: string) {
  return `${apiBaseURL}${path}`;
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(apiURL(path), init);
  let payload: ApiResponse<T>;
  try {
    payload = await response.json() as ApiResponse<T>;
  } catch {
    throw new ApiError("服务器返回了无法解析的响应", response.status, response.status);
  }
  if (!response.ok || payload.code !== 0) {
    throw new ApiError(payload.msg || "请求失败", payload.code, response.status);
  }
  return payload.data;
}

export async function requestBlob(path: string): Promise<Blob> {
  const response = await fetch(apiURL(path));
  if (!response.ok) {
    let message = "文件下载失败";
    try {
      const payload = await response.json() as ApiResponse<null>;
      message = payload.msg || message;
    } catch {
      // 二进制接口异常时可能没有 JSON 响应体。
    }
    throw new ApiError(message, response.status, response.status);
  }
  return response.blob();
}

export function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : "请求失败，请稍后重试。";
}
