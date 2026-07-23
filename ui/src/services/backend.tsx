import { Global_Data } from "@/config/config"
import axios, { isAxiosError, type AxiosRequestConfig } from "axios"

const api = axios.create({
  timeout: 120000,
})

api.interceptors.request.use((config) => {
  const token = Global_Data.getAccessToken()
  if (token) {
    config.headers.Authorization = token
  }
  return config
})

export type BackendResult<T = any> = {
  result: string
  data?: T
  error?: string
}

export async function Backend_Request<T = any>(
  body: unknown,
  url: string,
  config?: AxiosRequestConfig,
): Promise<BackendResult<T>> {
  try {
    const res = await api.post(Global_Data.server_ip + url, body, config)
    const payload = res.data

    if (Array.isArray(payload)) {
      return { result: "ok", data: payload as T }
    }

    if (payload && typeof payload === "object") {
      const body = payload as BackendResult<T>
      if (body.result === "error") {
        return {
          result: "error",
          error: String(body.error ?? "Xatolik"),
          data: body.data,
        }
      }
      if (body.data !== undefined) {
        return { result: "ok", data: body.data as T }
      }
      if (body.result === "ok") {
        return { result: "ok", data: body.data as T }
      }
    }

    return { result: "ok", data: payload as T }
  } catch (error) {
    if (isAxiosError(error)) {
      return {
        result: "error",
        error: error.response?.data?.error || error.message || "Unknown error",
        data: error.response?.data?.data,
      }
    }
    return { result: "error", error: String(error) }
  }
}

export async function Backend_Request_File<T = any>(
  body: unknown,
  url: string,
): Promise<BackendResult<T>> {
  return Backend_Request<T>(body, url, {
    headers: { "Content-Type": "multipart/form-data" },
  })
}

export async function Backend_Request_Blob(body: unknown, url: string): Promise<BackendResult<Blob>> {
  try {
    const res = await api.post(Global_Data.server_ip + url, body, {
      responseType: "blob",
    })
    return { result: "ok", data: res.data }
  } catch (error) {
    if (isAxiosError(error)) {
      return {
        result: "error",
        error: error.response?.data?.error || error.message || "Unknown error",
      }
    }
    return { result: "error", error: String(error) }
  }
}

export function t3SerialWsUrl(): string {
  const token = encodeURIComponent(Global_Data.getAccessToken())
  const path = `/api/lines/t3/v2/serialcurrent/ws?token=${token}`

  if (Global_Data.server_ip) {
    const wsBase = Global_Data.server_ip.replace(/^http/i, "ws")
    return `${wsBase}${path}`
  }

  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
  return `${protocol}//${window.location.host}${path}`
}
