import { Injectable, Inject, NotFoundException, BadRequestException, Controller, Body, Delete, Get, Param, Patch, Post, UseGuards } from '@nestjs/common';
import { Pool } from 'pg';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';

// AI provider registry + CRUD (check.md 2026-08-30 #17): the admin page
// previously claimed "Pending Backend" with a disabled form — this is the
// backend that removes that lie. API keys are referenced by secret-manager
// key name only (api_key_ref), never stored here.
@Injectable()
export class AiProvidersService {
  constructor(@Inject('DB_POOL') private pool: Pool) {}

  async list() { return (await this.pool.query(`SELECT * FROM ai.ai_providers ORDER BY created_at DESC`)).rows; }

  async get(id: string) {
    const r = await this.pool.query(`SELECT * FROM ai.ai_providers WHERE id = $1`, [id]);
    if (r.rows.length === 0) throw new NotFoundException('provider_not_found');
    return r.rows[0];
  }

  async create(body: { name: string; provider: string; base_url: string; api_key_ref?: string; model?: string }) {
    if (!body.name || !body.provider || !body.base_url) throw new BadRequestException('name/provider/base_url required');
    const exists = await this.pool.query(`SELECT 1 FROM ai.ai_providers WHERE name = $1`, [body.name]);
    if (exists.rowCount) throw new BadRequestException('name_exists');
    const r = await this.pool.query(
      `INSERT INTO ai.ai_providers (name, provider, base_url, api_key_ref, model, enabled)
       VALUES ($1,$2,$3,$4,$5, COALESCE($6, false)) RETURNING *`,
      [body.name, body.provider, body.base_url, body.api_key_ref || null, body.model || null, false],
    );
    return r.rows[0];
  }

  async update(id: string, body: { name?: string; base_url?: string; api_key_ref?: string; model?: string; enabled?: boolean }) {
    const r = await this.pool.query(
      `UPDATE ai.ai_providers SET
         name = COALESCE($2, name), base_url = COALESCE($3, base_url),
         api_key_ref = COALESCE($4, api_key_ref), model = COALESCE($5, model),
         updated_at = now()
       WHERE id = $1 RETURNING *`,
      [id, body.name || null, body.base_url || null, body.api_key_ref || null, body.model || null],
    );
    if (r.rows.length === 0) throw new NotFoundException('provider_not_found');
    return r.rows[0];
  }

  async remove(id: string) {
    const r = await this.pool.query(`DELETE FROM ai.ai_providers WHERE id = $1 RETURNING id`, [id]);
    if (r.rows.length === 0) throw new NotFoundException('provider_not_found');
    return { deleted: true, id };
  }

  async toggle(id: string, enabled: boolean) {
    const r = await this.pool.query(`UPDATE ai.ai_providers SET enabled = $2, updated_at = now() WHERE id = $1 RETURNING *`, [id, enabled]);
    if (r.rows.length === 0) throw new NotFoundException('provider_not_found');
    return r.rows[0];
  }
}

@Controller('operations/ai/providers')
@UseGuards(JwtAuthGuard, AdminGuard)
export class AiProvidersController {
  constructor(private svc: AiProvidersService) {}

  @Get() list() { return this.svc.list(); }
  @Get(':id') get(@Param('id') id: string) { return this.svc.get(id); }
  @Post() create(@CurrentUser('sub') userId: string, @Body() body: any) { return this.svc.create(body); }
  @Patch(':id') update(@Param('id') id: string, @Body() body: any) { return this.svc.update(id, body); }
  @Post(':id/enable') enable(@Param('id') id: string) { return this.svc.toggle(id, true); }
  @Post(':id/disable') disable(@Param('id') id: string) { return this.svc.toggle(id, false); }
  @Delete(':id') remove(@Param('id') id: string) { return this.svc.remove(id); }
}