import { customInstance } from './axios-instance';
import type { paths, components } from '@/generated/schema';

export type ApiResponse<T extends keyof paths> = paths[T] extends { get: { responses: { 200: { content: { 'application/json': infer R } } } } }
  ? R
  : paths[T] extends { post: { responses: { 201: { content: { 'application/json': infer R } } } } }
  ? R
  : unknown;

export async function apiGet<T extends keyof paths>(url: T, params?: unknown): Promise<ApiResponse<T>> {
  const res = await customInstance.get(url as string, { params });
  return res.data as ApiResponse<T>;
}

export async function apiPost<T extends keyof paths>(url: T, body?: unknown): Promise<ApiResponse<T>> {
  const res = await customInstance.post(url as string, body);
  return res.data as ApiResponse<T>;
}

export async function apiPatch<T extends keyof paths>(url: T, body?: unknown): Promise<ApiResponse<T>> {
  const res = await customInstance.patch(url as string, body);
  return res.data as ApiResponse<T>;
}

export type Schemas = components['schemas'];
