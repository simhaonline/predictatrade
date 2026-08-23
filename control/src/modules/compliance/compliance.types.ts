export interface ClientTelemetry {
  user_agent?: string;
  language?: string;
  languages?: string[];
  timezone?: string;
  timezone_offset_minutes?: number;
  screen?: {
    width: number;
    height: number;
    avail_width: number;
    avail_height: number;
  };
  viewport?: {
    width: number;
    height: number;
  };
  device_pixel_ratio?: number;
  color_depth?: number;
  touch_points?: number;
  platform?: string;
  client_hints?: Record<string, string | boolean | number>;
}

export interface AuditEventInput {
  event_type: string;
  event_version?: number;
  user_id?: string;
  account_id?: string;
  session_id?: string;
  request_id?: string;
  correlation_id?: string;
  http_method?: string;
  endpoint?: string;
  http_status?: number;
  latency_ms?: number;
  client_ip?: string;
  proxy_chain?: string[];
  user_agent?: string;
  prediction_id?: string;
  risk_flags?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  client_telemetry?: ClientTelemetry;
  geo?: {
    country_code?: string;
    region?: string;
    city?: string;
    isp?: string;
    asn?: number;
    as_org?: string;
  };
}

export const COMPLIANCE_EVENT_TYPES = [
  'AUTH_LOGIN', 'AUTH_LOGIN_FAILED', 'AUTH_LOGOUT', 'AUTH_PASSWORD_CHANGE',
  'AUTH_MFA_SUCCESS', 'AUTH_MFA_FAILURE', 'SESSION_CREATED', 'SESSION_REVOKED',
  'ACCOUNT_CREATED', 'ACCOUNT_UPDATED', 'PREDICTION_CREATED', 'PREDICTION_UPDATED',
  'PREDICTION_CANCELLED', 'PREDICTION_EXECUTED', 'PREDICTION_RESULT_RECORDED',
  'API_ACCESS', 'RATE_LIMIT_TRIGGERED', 'SECURITY_ALERT', 'SUSPICIOUS_REQUEST',
  'PRIVACY_CONSENT_UPDATED', 'ADMIN_ACTION', 'ADMIN_AUDIT_VIEW', 'ADMIN_USER_LOOKUP',
  'TELEMETRY_RECEIVED',
] as const;
