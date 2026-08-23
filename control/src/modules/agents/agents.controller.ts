import { Controller, Get, UseGuards } from '@nestjs/common';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';

interface AgentsStatus {
  agents_connected: number;
  master_node_connected: boolean;
  snapshot_count: number;
  mt4_connected: number;
  mt5_connected: number;
  backend_reachable: boolean;
  timestamp?: string;
  server_time?: string;
}

@Controller('agents')
@UseGuards(JwtAuthGuard)
export class AgentsController {
  @Get('status')
  async status(): Promise<AgentsStatus> {
    const url =
      process.env.GO_ENGINE_AGENTS_URL ||
      'http://127.0.0.1:13081/api/v1/agents/status';
    try {
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 3000);
      const res = await fetch(url, { signal: controller.signal });
      clearTimeout(timeout);
      const data = (await res.json()) as Record<string, unknown>;
      return {
        agents_connected: Number(data.agents_connected ?? 0),
        master_node_connected: Boolean(data.master_node_connected ?? false),
        snapshot_count: Number(data.snapshot_count ?? 0),
        mt4_connected: Number(data.mt4_connected ?? 0),
        mt5_connected: Number(data.mt5_connected ?? 0),
        backend_reachable: true,
        timestamp: data.timestamp as string | undefined,
        server_time: data.server_time as string | undefined,
      };
    } catch {
      return {
        agents_connected: 0,
        master_node_connected: false,
        snapshot_count: 0,
        mt4_connected: 0,
        mt5_connected: 0,
        backend_reachable: false,
      };
    }
  }
}
