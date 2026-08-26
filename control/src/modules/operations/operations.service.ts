import { Injectable, Inject, Logger, BadRequestException, NotFoundException } from '@nestjs/common';
import { Pool } from 'pg';
import { DB_POOL } from '../../common/database.module';

@Injectable()
export class OperationsService {
  private readonly logger = new Logger(OperationsService.name);

  constructor(@Inject(DB_POOL) private pool: Pool) {}

  /** Get current platform operation states (active halts, pauses, etc.) */
  async getActiveOperations() {
    const r = await this.pool.query(
      `SELECT * FROM control.platform_operations WHERE status = 'ACTIVE' ORDER BY created_at DESC`
    );
    return r.rows;
  }

  /** Halt all trading — stops signal execution but maintains connection/auth */
  async haltTrading(actorId: string, reason: string) {
    return this.createOperation('HALT_TRADING', 'execution', null, actorId, reason);
  }

  /** Resume trading after halt */
  async resumeTrading(actorId: string, reason: string) {
    await this.pool.query(
      `UPDATE control.platform_operations SET status = 'REVERTED', reverted_at = now() 
       WHERE operation_type = 'HALT_TRADING' AND status = 'ACTIVE'`
    );
    // RESUME is an instantaneous action, not a persistent state — record as COMPLETED.
    return this.createOperation('RESUME_TRADING', 'execution', null, actorId, reason, 'COMPLETED');
  }

  /** Pause signal generation */
  async pauseSignals(actorId: string, reason: string) {
    return this.createOperation('PAUSE_SIGNALS', 'signal_engine', null, actorId, reason);
  }

  /** Resume signal generation */
  async resumeSignals(actorId: string, reason: string) {
    await this.pool.query(
      `UPDATE control.platform_operations SET status = 'REVERTED', reverted_at = now() 
       WHERE operation_type = 'PAUSE_SIGNALS' AND status = 'ACTIVE'`
    );
    // RESUME is an instantaneous action, not a persistent state — record as COMPLETED.
    return this.createOperation('RESUME_SIGNALS', 'signal_engine', null, actorId, reason, 'COMPLETED');
  }

  /** Enable a specific strategy */
  async enableStrategy(strategyId: string, actorId: string, reason: string) {
    await this.pool.query(
      `UPDATE control.platform_operations SET status = 'REVERTED', reverted_at = now() 
       WHERE target_type = 'strategy' AND target_id = $1 AND operation_type = 'DISABLE_STRATEGY' AND status = 'ACTIVE'`,
      [strategyId]
    );
    // ENABLE is an instantaneous action — record as COMPLETED.
    return this.createOperation('ENABLE_STRATEGY', 'strategy', strategyId, actorId, reason, 'COMPLETED');
  }

  /** Disable a specific strategy */
  async disableStrategy(strategyId: string, actorId: string, reason: string) {
    return this.createOperation('DISABLE_STRATEGY', 'strategy', strategyId, actorId, reason);
  }

  /** Get all AI models with their status */
  async listModels() {
    const r = await this.pool.query(
      `SELECT id, name, version, model_type, status, metrics, activated_at, created_at 
       FROM ai.models ORDER BY created_at DESC`
    );
    return r.rows;
  }

  /** Get training jobs */
  async listTrainingJobs() {
    const r = await this.pool.query(
      `SELECT id, model_id, job_name, status, metrics, started_at, completed_at, error_message, created_at 
       FROM ai.training_jobs ORDER BY created_at DESC LIMIT 50`
    );
    return r.rows;
  }

  /** Get inference history */
  async listInferenceHistory(limit = 50) {
    const r = await this.pool.query(
      `SELECT ih.id, ih.model_id, m.name as model_name, ih.model_version, 
              ih.feature_timestamp, ih.prediction_timestamp, ih.inference_latency_ms,
              ih.prediction, ih.confidence, ih.model_health, ih.stale_feature, ih.fallback_used,
              ih.signal_id, ih.created_at
       FROM ai.inference_history ih
       LEFT JOIN ai.models m ON m.id = ih.model_id
       ORDER BY ih.created_at DESC LIMIT $1`, [limit]
    );
    return r.rows;
  }

  /** Activate a model (deactivate others of same type) */
  async activateModel(modelId: string, actorId: string) {
    const model = await this.pool.query('SELECT model_type FROM ai.models WHERE id = $1', [modelId]);
    if (model.rows.length === 0) throw new NotFoundException('Model not found');

    await this.pool.query('BEGIN');
    await this.pool.query(
      `UPDATE ai.models SET status = 'INACTIVE', deactivated_at = now(), updated_at = now() 
       WHERE model_type = $1 AND status = 'ACTIVE'`, [model.rows[0].model_type]
    );
    await this.pool.query(
      `UPDATE ai.models SET status = 'ACTIVE', activated_at = now(), updated_at = now() WHERE id = $1`,
      [modelId]
    );
    await this.pool.query(
      `INSERT INTO control.platform_operations (operation_type, target_type, target_id, status, actor_id, reason)
       VALUES ('ENABLE_EXECUTION', 'ai_model', $1, 'ACTIVE', $2, 'Model activated')`,
      [modelId, actorId]
    );
    await this.pool.query('COMMIT');
    return { success: true, model_id: modelId };
  }

  /** Deactivate a model */
  async deactivateModel(modelId: string, actorId: string) {
    await this.pool.query(
      `UPDATE ai.models SET status = 'INACTIVE', deactivated_at = now(), updated_at = now() WHERE id = $1`,
      [modelId]
    );
    await this.pool.query(
      `INSERT INTO control.platform_operations (operation_type, target_type, target_id, status, actor_id, reason)
       VALUES ('DISABLE_EXECUTION', 'ai_model', $1, 'ACTIVE', $2, 'Model deactivated')`,
      [modelId, actorId]
    );
    return { success: true, model_id: modelId };
  }

  /** Get trading halt state */
  async getTradingState() {
    const haltActive = await this.pool.query(
      `SELECT 1 FROM control.platform_operations WHERE operation_type = 'HALT_TRADING' AND status = 'ACTIVE' LIMIT 1`
    );
    const pauseActive = await this.pool.query(
      `SELECT 1 FROM control.platform_operations WHERE operation_type = 'PAUSE_SIGNALS' AND status = 'ACTIVE' LIMIT 1`
    );
    const disabledStrategies = await this.pool.query(
      `SELECT target_id FROM control.platform_operations 
       WHERE operation_type = 'DISABLE_STRATEGY' AND target_type = 'strategy' AND status = 'ACTIVE'`
    );
    const activeModel = await this.pool.query(
      `SELECT id, name, version FROM ai.models WHERE status = 'ACTIVE' LIMIT 1`
    );

    // Compute active_strategies: all canonical strategies minus explicitly disabled ones.
    // The frontend dashboard and operations page read `active_strategies` to show
    // per-strategy Active/Inactive badges and the aggregate active count.
    const canonicalStrategies = ['STANDARD_SCALPING', 'ULTRA_SCALPING', 'STANDARD_SWING', 'TREND_SWING', 'MARNIE_FIB'];
    const disabledSet = new Set(disabledStrategies.rows.map((r: any) => r.target_id));
    const activeStrategies = canonicalStrategies.filter((s) => !disabledSet.has(s));

    // last_updated reflects the most recent platform_operations change.
    const lastUpdatedRow = await this.pool.query(
      `SELECT created_at FROM control.platform_operations ORDER BY created_at DESC LIMIT 1`
    );

    return {
      trading_halted: haltActive.rows.length > 0,
      signals_paused: pauseActive.rows.length > 0,
      disabled_strategies: disabledStrategies.rows.map((r: any) => r.target_id),
      active_strategies: activeStrategies,
      active_ai_model: activeModel.rows[0] || null,
      last_updated: lastUpdatedRow.rows.length > 0 ? lastUpdatedRow.rows[0].created_at : null,
    };
  }

  /** Clean up stale RESUME and ENABLE operations that were incorrectly left ACTIVE. */
  async cleanupStaleOperations() {
    const r = await this.pool.query(
      `UPDATE control.platform_operations SET status = 'COMPLETED', completed_at = now()
       WHERE operation_type IN ('RESUME_TRADING', 'RESUME_SIGNALS', 'ENABLE_STRATEGY')
         AND status = 'ACTIVE'`
    );
    if (r.rowCount > 0) {
      this.logger.log(`Cleaned up ${r.rowCount} stale instantaneous operations left ACTIVE`);
    }
    return r.rowCount;
  }

  private async createOperation(opType: string, targetType: string, targetId: string | null, actorId: string, reason: string, status = 'ACTIVE') {
    const r = await this.pool.query(
      `INSERT INTO control.platform_operations (operation_type, target_type, target_id, status, actor_id, reason, created_at)
       VALUES ($1, $2, $3, $4, $5, $6, now()) RETURNING *`,
      [opType, targetType, targetId, status, actorId, reason]
    );
    this.logger.log(`Operation ${opType} by ${actorId}: ${reason}`);
    // Audit the operation
    try {
      await this.pool.query(
        `INSERT INTO audit.audit_events (id, event_id, actor_type, actor_id, action, entity_type, entity_id, new_value, reason, timestamp)
         VALUES (gen_random_uuid(), gen_random_uuid(), 'USER', $1, $2, $3, $4, $5, $6, now())`,
        [actorId, opType, targetType, targetId, JSON.stringify({ status, reason }), reason]
      );
    } catch {
      this.logger.warn(`Failed to write audit event for operation ${opType}`);
    }
    return r.rows[0];
  }
}
