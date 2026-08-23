import { customInstance } from "@/lib/axios-instance";

export interface AIModel {
  id: string;
  name: string;
  version?: string;
  model_type?: string;
  status?: "ACTIVE" | "INACTIVE" | "TRAINING" | "ARCHIVED" | string;
  metrics?: Record<string, unknown>;
  activated_at?: string;
  created_at?: string;
}

export async function fetchAIModels(): Promise<AIModel[]> {
  const res = await customInstance.get("/operations/ai/models");
  return Array.isArray(res.data) ? (res.data as AIModel[]) : [];
}

export async function activateModel(id: string) {
  const res = await customInstance.post(`/operations/ai/model/${id}/activate`);
  return res.data;
}

export async function deactivateModel(id: string) {
  const res = await customInstance.post(`/operations/ai/model/${id}/deactivate`);
  return res.data;
}
