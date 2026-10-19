import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse, AxiosError } from "axios";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

const TOKEN_KEY = "auth_token";

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  if (typeof window === "undefined") return;
  localStorage.removeItem(TOKEN_KEY);
}

export interface ApiError {
  code: number;
  message: string;
  details?: unknown;
}

export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

class ApiClient {
  private instance: AxiosInstance;

  constructor(baseURL: string) {
    this.instance = axios.create({
      baseURL,
      timeout: 30000,
      headers: {
        "Content-Type": "application/json",
      },
    });

    // Request interceptor
    this.instance.interceptors.request.use(
      (config) => {
        const token = getToken();
        if (token && config.headers) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response interceptor
    this.instance.interceptors.response.use(
      (response: AxiosResponse<ApiResponse<unknown>>) => {
        // Check if response has the expected structure
        if (response.data && typeof response.data.code === "number") {
          if (response.data.code !== 0) {
            const apiError: ApiError = {
              code: response.data.code,
              message: response.data.message || "Request failed",
              details: response.data.data,
            };
            return Promise.reject(apiError);
          }
        }
        return response;
      },
      (error: AxiosError<ApiResponse<unknown>>) => {
        if (error.response?.status === 401) {
          clearToken();
          if (typeof window !== "undefined") {
            window.location.href = "/login";
          }
          const apiError: ApiError = {
            code: 401,
            message: "Unauthorized",
          };
          return Promise.reject(apiError);
        }

        const apiError: ApiError = {
          code: error.response?.status || 500,
          message: error.response?.data?.message || error.message || "Network error",
          details: error.response?.data,
        };
        return Promise.reject(apiError);
      }
    );
  }

  private async request<T>(config: AxiosRequestConfig): Promise<T> {
    const response = await this.instance.request<ApiResponse<T>>(config);
    return response.data.data;
  }

  get<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return this.request<T>({ ...config, method: "GET", url });
  }

  post<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    return this.request<T>({ ...config, method: "POST", url, data });
  }

  put<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    return this.request<T>({ ...config, method: "PUT", url, data });
  }

  delete<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return this.request<T>({ ...config, method: "DELETE", url });
  }

  patch<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> {
    return this.request<T>({ ...config, method: "PATCH", url, data });
  }
}

export const api = new ApiClient(API_BASE);

// Legacy support for direct fetch usage
export async function fetchApi<T>(
  endpoint: string,
  options?: RequestInit
): Promise<ApiResponse<T>> {
  const token = getToken();

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });

  const data = await response.json();

  if (response.status === 401) {
    clearToken();
    if (typeof window !== "undefined") {
      window.location.href = "/login";
    }
  }

  return data;
}
